#!/bin/bash

# Check OS
OS=$(cat /etc/os-release | grep ^ID= | cut -d'=' -f2)
temp="${OS%\"}"
temp="${temp#\"}"
OS="$temp"

SERVICE_USERNAME=ihub
SERVICE_ENV=ihub.env

# READ .env file
echo PWD IS $(pwd)
if [ -f ~/$SERVICE_ENV ]; then
    echo Reading Installation options from $(realpath ~/$SERVICE_ENV)
    env_file=~/$SERVICE_ENV
elif [ -f ../$SERVICE_ENV ]; then
    echo Reading Installation options from $(realpath ../$SERVICE_ENV)
    env_file=../$SERVICE_ENV
fi

if [ -z $env_file ]; then
    echo No .env file found
    IHUB_NOSETUP="true"
else
    source $env_file
    env_file_exports=$(cat $env_file | grep -E '^[A-Z0-9_]+\s*=' | cut -d = -f 1)
    if [ -n "$env_file_exports" ]; then eval export $env_file_exports; fi
fi

COMPONENT_NAME=$SERVICE_USERNAME
INSTANCE_NAME=${INSTANCE_NAME:-$COMPONENT_NAME}
SERVICE_NAME=$SERVICE_USERNAME@$INSTANCE_NAME

service_exists() {
    if [[ $(systemctl list-units --all -t service --full --no-legend "$1.service" | cut -f1 -d' ') == $1.service ]]; then
        return 0
    else
        return 1
    fi
}

# Upgrade if service is already installed
if service_exists $SERVICE_USERNAME || service_exists $SERVICE_NAME; then
  n=0
  until [ "$n" -ge 3 ]
  do
  echo "$COMPONENT_NAME is already installed, Do you want to proceed with the upgrade? [y/n]"
  read UPGRADE_NEEDED
  if [ $UPGRADE_NEEDED == "y" ] || [ $UPGRADE_NEEDED == "Y" ] ; then
    echo "Proceeding with the upgrade.."
    ./${COMPONENT_NAME}_upgrade.sh
    exit $?
  elif [ $UPGRADE_NEEDED == "n" ] || [ $UPGRADE_NEEDED == "N" ] ; then
    echo "Exiting the installation.."
    exit 0
  fi
  n=$((n+1))
  done
  echo "Exiting the installation.."
  exit 0
fi

if [[ $EUID -ne 0 ]]; then
    echo "This installer must be run as root"
    exit 1
fi

echo "Setting up IHUB Linux User..."
# useradd -M -> this user has no home directory
id -u $SERVICE_USERNAME 2> /dev/null || useradd -M --system --shell /sbin/nologin $SERVICE_USERNAME

echo "Installing Integration Hub Service..."

PRODUCT_HOME=/opt/$COMPONENT_NAME
BIN_PATH=$PRODUCT_HOME/bin
LOG_PATH=/var/log/$INSTANCE_NAME/
CONFIG_PATH=/etc/$INSTANCE_NAME/
CERTS_PATH=$CONFIG_PATH/certs
CERTDIR_TRUSTEDJWTCAS=$CERTS_PATH/trustedca
SAML_CERT_DIR_PATH=$CERTS_PATH/saml

for directory in $BIN_PATH $LOG_PATH $CONFIG_PATH $CERTS_PATH $CERTDIR_TRUSTEDJWTCAS $SAML_CERT_DIR_PATH; do
    mkdir -p $directory
    if [ $? -ne 0 ]; then
        echo "Cannot create directory: $directory"
        exit 1
    fi
    chown -R $SERVICE_USERNAME:$SERVICE_USERNAME $directory
    chmod 700 $directory
done

cp $COMPONENT_NAME $BIN_PATH/ && chown $SERVICE_USERNAME:$SERVICE_USERNAME $BIN_PATH/*
chmod 700 $BIN_PATH/*
ln -sfT $BIN_PATH/$COMPONENT_NAME /usr/bin/$COMPONENT_NAME

# log file permission change
chmod 740 $LOG_PATH

# Install systemd script
SERVICE_FILE=$SERVICE_USERNAME@.service
cp $SERVICE_USERNAME.service $PRODUCT_HOME/$SERVICE_FILE && chown $SERVICE_USERNAME:$SERVICE_USERNAME $PRODUCT_HOME/$SERVICE_FILE && chown $SERVICE_USERNAME:$SERVICE_USERNAME $PRODUCT_HOME

# Enable systemd service
systemctl disable $PRODUCT_HOME/$SERVICE_FILE >/dev/null 2>&1
systemctl enable $PRODUCT_HOME/$SERVICE_FILE
systemctl disable $COMPONENT_NAME@$INSTANCE_NAME >/dev/null 2>&1
systemctl enable $COMPONENT_NAME@$INSTANCE_NAME
systemctl daemon-reload

auto_install() {
  local component=${1}
  local cprefix=${2}
  local packages=$(eval "echo \$${cprefix}_PACKAGES")
  # detect available package management tools. start with the less likely ones to differentiate.
if [ "$OS" == "rhel" ]
then
  yum -y install $packages
elif [ "$OS" == "ubuntu" ]
then
  apt -y install $packages
fi
}

# SCRIPT EXECUTION
logRotate_clear() {
  logrotate=""
}

logRotate_detect() {
  local logrotaterc=`ls -1 /etc/logrotate.conf 2>/dev/null | tail -n 1`
  logrotate=`which logrotate 2>/dev/null`
  if [ -z "$logrotate" ] && [ -f "/usr/sbin/logrotate" ]; then
    logrotate="/usr/sbin/logrotate"
  fi
}

logRotate_install() {
  LOGROTATE_PACKAGES="logrotate"
  if [ "$(whoami)" == "root" ]; then
    auto_install "Log Rotate" "LOGROTATE"
    if [ $? -ne 0 ]; then echo "Failed to install logrotate"; exit -1; fi
  fi
  logRotate_clear; logRotate_detect;
    if [ -z "$logrotate" ]; then
      echo "logrotate is not installed"
    else
      echo  "logrotate installed in $logrotate"
    fi
}

logRotate_install

export LOG_ROTATION_PERIOD=${LOG_ROTATION_PERIOD:-weekly}
export LOG_COMPRESS=${LOG_COMPRESS:-compress}
export LOG_DELAYCOMPRESS=${LOG_DELAYCOMPRESS:-delaycompress}
export LOG_COPYTRUNCATE=${LOG_COPYTRUNCATE:-copytruncate}
export LOG_SIZE=${LOG_SIZE:-100M}
export LOG_OLD=${LOG_OLD:-12}

mkdir -p /etc/logrotate.d

if [ ! -a /etc/logrotate.d/${INSTANCE_NAME} ]; then
 echo "/var/log/${INSTANCE_NAME}/*.log {
    missingok
    notifempty
    rotate $LOG_OLD
    maxsize $LOG_SIZE
    nodateext
    $LOG_ROTATION_PERIOD
    $LOG_COMPRESS
    $LOG_DELAYCOMPRESS
    $LOG_COPYTRUNCATE
}" > /etc/logrotate.d/${INSTANCE_NAME}
fi


# check if IHUB_NOSETUP is defined
if [ "${IHUB_NOSETUP,,}" == "true" ]; then
    echo "IHUB_NOSETUP is true, skipping setup"
    echo "Run \"$COMPONENT_NAME setup all\" for manual setup"
    echo "Installation completed successfully!"
else    
    $COMPONENT_NAME setup all --force -i $INSTANCE_NAME
    SETUPRESULT=$?
    chown -R $SERVICE_USERNAME:$SERVICE_USERNAME $CONFIG_PATH
    if [ ${SETUPRESULT} == 0 ]; then
        systemctl start $SERVICE_NAME
        echo "Waiting for daemon to settle down before checking status"
        sleep 3
        systemctl status $SERVICE_NAME 2>&1 >/dev/null
        if [ $? != 0 ]; then
          echo "Installation completed with Errors - $SERVICE_NAME daemon not started."
          echo "Please check errors in syslog using \`journalctl -u $SERVICE_NAME\`"
          exit 1
        fi
        echo "$SERVICE_NAME daemon is running"
        echo "Installation completed successfully!"
    else
        echo "Installation completed with errors"
    fi
fi
