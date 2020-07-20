/*
 * Copyright (C) 2020 Intel Corporation
 * SPDX-License-Identifier: BSD-3-Clause
 */

package postgres

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"github.com/intel-secl/intel-secl/v3/pkg/hvs/domain/models"
	commErr "github.com/intel-secl/intel-secl/v3/pkg/lib/common/err"
	"github.com/intel-secl/intel-secl/v3/pkg/model/hvs"
	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"time"
)

type ReportStore struct {
	Store *DataStore
}

func NewReportStore(store *DataStore) *ReportStore {
	return &ReportStore{store}
}

// Retrieve method fetches report for a given Id
func (r *ReportStore) Retrieve(reportId uuid.UUID) (*models.HVSReport, error) {
	defaultLog.Trace("postgres/report_store:Retrieve() Entering")
	defer defaultLog.Trace("postgres/report_store:Retrieve() Leaving")

	re := models.HVSReport{}

	row := r.Store.Db.Model(&report{}).Where(&report{ID: reportId}).Row()
	if err := row.Scan(&re.ID, &re.HostID, (*PGTrustReport)(&re.TrustReport), &re.CreatedAt, &re.Expiration, &re.Saml); err != nil {
		return nil, errors.Wrap(err, "postgres/report_store:Retrieve() failed to scan record")
	}

	return &re, nil
}

// Update method is called after completion of flavor verification process by the flavor verify queue
func (r *ReportStore) Update(re *models.HVSReport) (*models.HVSReport, error) {
	defaultLog.Trace("postgres/report_store:Update() Entering")
	defer defaultLog.Trace("postgres/report_store:Update() Leaving")

	var refilter models.ReportFilterCriteria
	if re.HostID == uuid.Nil {
		return nil, errors.New("Host ID must be specified")
	} else {
		refilter = models.ReportFilterCriteria{
			HostID:         re.HostID,
			LatestPerHost:  true,
		}
	}

	var vsReport *models.HVSReport
	hvsReports, err := r.Search(&refilter)
	if err != nil {
		if strings.Contains(err.Error(), commErr.RowsNotFound) {
			vsReport, err = r.Create(re)
			if err != nil {
				return nil, errors.Wrap(err, "postgres/report_store:Update() Error while creating report")
			}
		} else {
			return nil, errors.Wrap(err, "postgres/report_store:Update() Error while searching report")
		}
	}

	//TODO: Remove this once error is thrown
	if len(hvsReports) == 0 {
		vsReport, err = r.Create(re)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/report_store:Update() Error while creating report")
		}
	} else {
		vsReport = hvsReports[0]
	}

	dbReport := report{}
	if vsReport.HostID != uuid.Nil {
		dbReport.HostID = vsReport.HostID
	}
	if !vsReport.CreatedAt.IsZero() {
		dbReport.CreatedAt = vsReport.CreatedAt
	}
	if !vsReport.Expiration.IsZero() {
		dbReport.Expiration = vsReport.Expiration
	}
	if !reflect.DeepEqual(vsReport.TrustReport, hvs.TrustReport{}) {
		dbReport.TrustReport = PGTrustReport(vsReport.TrustReport)
	}
	if vsReport.Saml != "" {
		dbReport.Saml = vsReport.Saml
	}

	if db := r.Store.Db.Model(&dbReport).Updates(&dbReport); db.Error != nil || db.RowsAffected != 1 {
		if db.Error != nil {
			return nil, errors.Wrap(db.Error, "postgres/report_store:Update() failed to update HVSReport  "+dbReport.ID.String())
		} else {
			return nil, errors.New("postgres/report_store:Update() - rows affected with Id = %s" + dbReport.ID.String())
		}
	}

	return vsReport, nil
}

// Create method creates a new record in report table
func (r *ReportStore) Create(re *models.HVSReport) (*models.HVSReport, error) {
	defaultLog.Trace("postgres/report_store:Create() Entering")
	defer defaultLog.Trace("postgres/report_store:Create() Leaving")

	re.ID = uuid.New()
	dbReport := report{
		ID:          re.ID,
		HostID:      re.HostID,
		CreatedAt:   re.CreatedAt,
		Expiration:  re.Expiration,
		Saml:        re.Saml,
		TrustReport: PGTrustReport(re.TrustReport),
	}
	if err := r.Store.Db.Create(&dbReport).Error; err != nil {
		return nil, errors.Wrap(err, "postgres/report_store:Create() failed to create HVSReport")
	}

	return re, nil
}

// Search retrieves collection of HVSReport pertaining to a user-provided ReportFilterCriteria
func (r *ReportStore) Search(criteria *models.ReportFilterCriteria) ([]*models.HVSReport, error) {
	defaultLog.Trace("postgres/report_store:Search() Entering")
	defer defaultLog.Trace("postgres/report_store:Search() Leaving")

	var reportID uuid.UUID
	var hostID uuid.UUID
	var hostName string
	var hostHardwareUUID uuid.UUID
	var hostStatus string
	var latestPerHost bool
	var toDate time.Time
	var fromDate time.Time

	if criteria.ID != uuid.Nil {
		reportID = criteria.ID
	}
	if criteria.HostID != uuid.Nil {
		hostID = criteria.HostID
	}
	if criteria.HostHardwareID != uuid.Nil {
		hostHardwareUUID = criteria.HostHardwareID
	}
	if criteria.HostStatus != "" {
		hostStatus = criteria.HostStatus
	}
	if criteria.HostName != "" {
		hostName = criteria.HostName
	}
	if !criteria.ToDate.IsZero() {
		toDate = criteria.ToDate
	}
	if !criteria.FromDate.IsZero() {
		fromDate = criteria.FromDate
	}
	if criteria.Limit == 0 && criteria.LatestPerHost {
		criteria.Limit = 1
	} else {
		criteria.Limit = 2000
	}
	latestPerHost = criteria.LatestPerHost

	if criteria.NumberOfDays != 0 {
		toDate = time.Now()
		fromDate = toDate.AddDate(0, 0, -(criteria.NumberOfDays))
	}
	var tx *gorm.DB
	if criteria.FromDate.IsZero() && criteria.ToDate.IsZero() && criteria.LatestPerHost {
		tx = buildLatestReportSearchQuery(r.Store.Db, reportID, hostID, hostHardwareUUID, hostName, hostStatus, criteria.Limit)

		if tx == nil {
			return nil, errors.New("postgres/report_store:Search() Unexpected Error. Could not build" +
				" a gorm query object in HVSReport Search function.")
		}

		rows, err := tx.Rows()
		if err != nil {
			return nil, errors.Wrap(err, "postgres/report_store:Search() failed to retrieve records from db")
		}
		defer rows.Close()

		reports := []*models.HVSReport{}

		for rows.Next() {
			result := models.HVSReport{}

			if err := rows.Scan(&result.ID, &result.HostID, (*PGTrustReport)(&result.TrustReport), &result.CreatedAt, &result.Expiration, &result.Saml); err != nil {
				return nil, errors.Wrap(err, "postgres/report_store:Search() failed to scan record")
			}
			reports = append(reports, &result)
		}

		return reports, nil
	} else {
		tx = buildReportSearchQuery(r.Store.Db, hostID, hostHardwareUUID, hostName, hostStatus, fromDate, toDate, latestPerHost, criteria.Limit)
		if tx == nil {
			return nil, errors.New("postgres/report_store:Search() Unexpected Error. Could not build" +
				" a gorm query object in HVSReport Search function.")
		}

		rows, err := tx.Rows()
		if err != nil {
			return nil, errors.Wrap(err, "postgres/report_store:Search() failed to retrieve records from db")
		}
		defer rows.Close()

		reports := []*models.HVSReport{}
		for rows.Next() {
			result := models.AuditLogEntry{}
			if err := rows.Scan(&result.ID, &result.EntityID, &result.EntityType, &result.CreatedAt, &result.Action, (*PGAuditLogData)(&result.Data)); err != nil {
				return nil, errors.Wrap(err, "postgres/report_store:Search() failed to scan record")
			}
			if reflect.DeepEqual(models.AuditTableData{}, result.Data) || len(result.Data.Columns) == 0 {
				continue
			}
			hvsReport, err := auditlogEntryToReport(result)
			if err != nil {
				return nil, errors.Wrap(err, "postgres/report_store:Search() convert auditloag entry into report")
			}
			reports = append(reports, hvsReport)
		}

		return reports, nil
	}
}

// FindHostIdsFromExpiredReports searches the report table for reports that have an
// 'expiration' between 'fromTime' and 'toTime'.
func (r *ReportStore) FindHostIdsFromExpiredReports(fromTime time.Time, toTime time.Time) ([]uuid.UUID, error) {

	// TODO: https://jira.devtools.intel.com/browse/ISECL-10985
	query := "select h.id from host as h where exists (select t.host_id from (select row_number() over (partition by host_id order by expiration desc) rn, host_id from report where expiration > CAST(? AS TIMESTAMP) and expiration < CAST(? AS TIMESTAMP)) as t where h.id=t.host_id and t.rn=1);"
	
	hostIDs := []uuid.UUID{}
	err := r.Store.Db.Raw(query, fromTime, toTime).Scan(&hostIDs).Error
	if err != nil {
		return nil, err
	}

	return hostIDs, nil
}

func auditlogEntryToReport(auRecord models.AuditLogEntry) (*models.HVSReport, error) {
	defaultLog.Trace("postgres/report_store:auditlogEntryToReport() Entering")
	defer defaultLog.Trace("postgres/report_store:auditlogEntryToReport() Leaving")

	var hvsReport models.HVSReport

	if auRecord.EntityID != uuid.Nil {
		hvsReport.ID = auRecord.EntityID
	}
	// TODO remove duplicate data: first column and the entityID are both same
	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[1]) && auRecord.Data.Columns[1].Value != nil {
		hvsReport.HostID = uuid.MustParse(fmt.Sprintf("%v", auRecord.Data.Columns[1].Value))
	}

	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[2]) && auRecord.Data.Columns[2].Value != nil {
		c, err := json.Marshal(auRecord.Data.Columns[2].Value)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToReport() - marshalling failed")
		}
		err = json.Unmarshal(c, &hvsReport.TrustReport)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToReport() - unmarshalling failed")
		}
	}

	var err error
	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[3]) && auRecord.Data.Columns[2].Value != nil {
		createdString := fmt.Sprintf("%v", auRecord.Data.Columns[3].Value)
		//TODO use standard UTC time in auditlog handler while inserting time from reports.
		hvsReport.CreatedAt, err = time.Parse("2006-01-02T15:04:05-0700", createdString)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToReport() - error parsing time")
		}
	}

	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[4]) && auRecord.Data.Columns[4].Value != nil {
		expString := fmt.Sprintf("%v", auRecord.Data.Columns[4].Value)
		//TODO use standard UTC time in auditlog handler while inserting time from reports.
		hvsReport.Expiration, err = time.Parse("2006-01-02T15:04:05-0700", expString)
		if err != nil {
			return nil, errors.Wrap(err, "postgres/reports_store:auditlogEntryToReport() - error parsing time")
		}
	}

	if !reflect.DeepEqual(models.AuditColumnData{}, auRecord.Data.Columns[5]) && auRecord.Data.Columns[5].Value != nil {
		hvsReport.Saml = fmt.Sprintf("%v", auRecord.Data.Columns[5].Value)
	}

	return &hvsReport, nil
}

// Delete method deletes report for a given Id
func (r *ReportStore) Delete(reportId uuid.UUID) error {
	defaultLog.Trace("postgres/report_store:Delete() Entering")
	defer defaultLog.Trace("postgres/report_store:Delete() Leaving")

	if err := r.Store.Db.Delete(&report{ID: reportId}).Error; err != nil {
		return errors.Wrap(err, "postgres/report_store:Delete() failed to delete Report")
	}
	return nil
}

// buildReportSearchQuery is a helper function to build the query object for a report search.
func buildReportSearchQuery(tx *gorm.DB, hostHardwareID, hostID uuid.UUID, hostName, hostState string, fromDate, toDate time.Time, latestPerHost bool, limit int) *gorm.DB {
	defaultLog.Trace("postgres/report_store:buildReportSearchQuery() Entering")
	defer defaultLog.Trace("postgres/report_store:buildReportSearchQuery() Leaving")

	if tx == nil {
		return nil
	}
	//TODO remove after feature development
	tx.LogMode(true)
	if latestPerHost {
		entity := "auj"
		txSubQuery := tx.Table("audit_log_entry auj").Select("entity_id, max(auj.created) AS max_date")
		txSubQuery = buildReportSearchQueryWithCriteria(txSubQuery, hostHardwareID, hostID, entity, hostName, hostState, fromDate, toDate)
		txSubQuery = txSubQuery.Group("entity_id")
		subQuery := txSubQuery.SubQuery()
		tx = tx.Table("audit_log_entry au").Select("au.*").Joins("INNER JOIN ? a ON a.entity_id = au.entity_id AND a.max_date = au.created", subQuery)
	} else {
		entity := "au"
		tx = tx.Table("audit_log_entry au").Select("au.*")
		tx = buildReportSearchQueryWithCriteria(tx, hostHardwareID, hostID, entity, hostName, hostState, fromDate, toDate)
	}
	tx = tx.Limit(limit)
	return tx
}

func buildReportSearchQueryWithCriteria(tx *gorm.DB, hostHardwareID, hostID uuid.UUID, entity, hostName string, hostState string, fromDate, toDate time.Time) *gorm.DB {
	defaultLog.Trace("postgres/report_store:buildReportSearchQueryWithCriteria() Entering")
	defer defaultLog.Trace("postgres/report_store:buildReportSearchQueryWithCriteria() Leaving")

	if hostState != "" {
		tx = tx.Joins("INNER JOIN host_status hs on CAST(hs.host_id AS VARCHAR) = " + entity + ".data -> 'columns' -> 1 ->> 'value'")
	}

	if hostName != "" || hostHardwareID != uuid.Nil {
		tx = tx.Joins("INNER JOIN host h on CAST(h.id AS VARCHAR) = " + entity + ".data -> 'columns' -> 1 ->> 'value'")
	}

	//TODO rename after testing
	tx = tx.Where(entity + ".entity_type = 'Report'")

	if hostName != "" {
		tx = tx.Where("h.name = ?", hostName)
	}

	if hostHardwareID != uuid.Nil {
		tx = tx.Where("h.hardware_uuid = ?", hostHardwareID.String())
	}

	if hostID != uuid.Nil {
		tx = tx.Where(entity+".data -> 'columns' -> 1 ->> 'value' = ?", hostID.String())
	}

	if hostState != "" {
		tx = tx.Where("hs.status ->> 'host_state' = ?", strings.ToUpper(hostState))
	}

	if !fromDate.IsZero() {
		tx = tx.Where("CAST("+entity+".created AS TIMESTAMP) >= CAST(? AS TIMESTAMP)", fromDate)
	}

	if !toDate.IsZero() {
		tx = tx.Where("CAST("+entity+".created AS TIMESTAMP) < CAST(? AS TIMESTAMP)", toDate)
	}

	return tx
}

// buildLatestReportSearchQuery is a helper function to build the query object for a latest report search.
func buildLatestReportSearchQuery(tx *gorm.DB, reportID, hostID, hostHardwareID  uuid.UUID, hostName, hostState string, limit int) *gorm.DB {
	defaultLog.Trace("postgres/report_store:buildLatestReportSearchQuery() Entering")
	defer defaultLog.Trace("postgres/report_store:buildLatestReportSearchQuery() Leaving")

	if tx == nil {
		return nil
	}
	//TODO remove after feature development
	tx.LogMode(true)
	tx = tx.Model(&report{})

	// Since report id is unique and only one record can be returned by the query.
	if reportID != uuid.Nil {
		tx = tx.Where("id = '%s'", reportID.String())
		return tx
	}

	if  hostID != uuid.Nil || hostName != "" || hostHardwareID != uuid.Nil {
		tx = tx.Joins("INNER JOIN host h on h.id = host_id")
	}

	if hostState != "" {
		tx = tx.Joins("INNER JOIN host_status hs on hs.host_id = report.host_id")
		tx = tx.Where(`hs.status ->> 'host_state'=?`, strings.ToUpper(hostState))
	}

	if hostName != "" {
		tx = tx.Where("h.name = ?", hostName)
	}

	if hostHardwareID != uuid.Nil {
		tx = tx.Where("h.hardware_uuid = ?", hostHardwareID.String())
	}

	if hostID != uuid.Nil {
		tx = tx.Where("host_id = ?", hostID.String())
	}

	tx = tx.Limit(limit)
	return tx
}
