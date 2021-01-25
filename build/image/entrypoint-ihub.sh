#!/bin/bash

USER_ID=$(id -u)
SERVICE_USERNAME=ihub
PRODUCT_HOME=/opt/$SERVICE_USERNAME
BIN_PATH=$PRODUCT_HOME/bin
LOG_PATH=/var/log/$SERVICE_USERNAME/
CONFIG_PATH=/etc/$SERVICE_USERNAME/
CERTS_PATH=$CONFIG_PATH/certs
CERTDIR_TRUSTEDJWTCAS=$CERTS_PATH/trustedca
SAML_CERT_DIR_PATH=$CERTS_PATH/saml

if [ ! -f $CONFIG_PATH/.setup_done ]; then
  for directory in $PRODUCT_HOME $BIN_PATH $LOG_PATH $CONFIG_PATH $CERTS_PATH $CERTDIR_TRUSTEDJWTCAS $SAML_CERT_DIR_PATH; do
    mkdir -p $directory
    if [ $? -ne 0 ]; then
      echo "Cannot create directory: $directory"
      exit 1
    fi
    chown -R $USER_ID:$USER_ID $directory
    chmod 700 $directory
  done
  ihub setup all
  if [ $? -ne 0 ]; then
    exit 1
  fi
  touch $CONFIG_PATH/.setup_done
fi

if [ ! -z $SETUP_TASK ]; then
  IFS=',' read -ra ADDR <<< "$SETUP_TASK"
  for task in "${ADDR[@]}"; do
    ihub setup $task --force
    if [ $? -ne 0 ]; then
      exit 1
    fi
  done
fi

ihub run
