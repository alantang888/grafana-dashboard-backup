package main

import (
	"encoding/json"
	"github.com/prometheus/common/model"
	"time"
)

// This is copy from https://github.com/grafana/grafana
// Since that repo have `replace directives`. Can't add to `go.sum`.
// /pkg/services/ngalert/models/provisioning.go
type Provenance string

const (
	// ProvenanceNone reflects the provenance when no provenance is stored
	// for the requested object in the database.
	ProvenanceNone Provenance = ""
	ProvenanceAPI  Provenance = "api"
	ProvenanceFile Provenance = "file"
)

// pkg/services/ngalert/models/alert_query.go
type RelativeTimeRange struct {
	From time.Duration `json:"from" yaml:"from"`
	To   time.Duration `json:"to" yaml:"to"`
}

type AlertQuery struct {
	// RefID is the unique identifier of the query, set by the frontend call.
	RefID string `json:"refId"`

	// QueryType is an optional identifier for the type of query.
	// It can be used to distinguish different types of queries.
	QueryType string `json:"queryType"`

	// RelativeTimeRange is the relative Start and End of the query as sent by the frontend.
	RelativeTimeRange RelativeTimeRange `json:"relativeTimeRange"`

	// Grafana data source unique identifier; it should be '__expr__' for a Server Side Expression operation.
	DatasourceUID string `json:"datasourceUid"`

	// JSON is the raw JSON query and includes the above properties as well as custom properties.
	Model json.RawMessage `json:"model"`

	modelProps map[string]interface{}
}

// /pkg/services/ngalert/api/tooling/definitions/cortex-ruler.go
type NoDataState string

const (
	Alerting NoDataState = "Alerting"
	NoData   NoDataState = "NoData"
	OK       NoDataState = "OK"
)

type ExecutionErrorState string

const (
	OkErrState       ExecutionErrorState = "OK"
	AlertingErrState ExecutionErrorState = "Alerting"
	ErrorErrState    ExecutionErrorState = "Error"
)

type NamespaceConfigResponse map[string][]GettableRuleGroupConfig

type GettableRuleGroupConfig struct {
	Name          string                     `yaml:"name" json:"name"`
	Interval      model.Duration             `yaml:"interval,omitempty" json:"interval,omitempty"`
	SourceTenants []string                   `yaml:"source_tenants,omitempty" json:"source_tenants,omitempty"`
	Rules         []GettableExtendedRuleNode `yaml:"rules" json:"rules"`
}

type GettableExtendedRuleNode struct {
	// note: this works with yaml v3 but not v2 (the inline tag isn't accepted on pointers in v2)
	*ApiRuleNode `yaml:",inline"`
	//GrafanaManagedAlert yaml.Node `yaml:"grafana_alert,omitempty"`
	GrafanaManagedAlert *GettableGrafanaRule `yaml:"grafana_alert,omitempty" json:"grafana_alert,omitempty"`
}

type ApiRuleNode struct {
	Record      string            `yaml:"record,omitempty" json:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty" json:"alert,omitempty"`
	Expr        string            `yaml:"expr" json:"expr"`
	For         *model.Duration   `yaml:"for,omitempty" json:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty" json:"annotations,omitempty"`
}

type GettableGrafanaRule struct {
	ID              int64               `json:"id" yaml:"id"`
	OrgID           int64               `json:"orgId" yaml:"orgId"`
	Title           string              `json:"title" yaml:"title"`
	Condition       string              `json:"condition" yaml:"condition"`
	Data            []AlertQuery        `json:"data" yaml:"data"`
	Updated         time.Time           `json:"updated" yaml:"updated"`
	IntervalSeconds int64               `json:"intervalSeconds" yaml:"intervalSeconds"`
	Version         int64               `json:"version" yaml:"version"`
	UID             string              `json:"uid" yaml:"uid"`
	NamespaceUID    string              `json:"namespace_uid" yaml:"namespace_uid"`
	NamespaceID     int64               `json:"namespace_id" yaml:"namespace_id"`
	RuleGroup       string              `json:"rule_group" yaml:"rule_group"`
	NoDataState     NoDataState         `json:"no_data_state" yaml:"no_data_state"`
	ExecErrState    ExecutionErrorState `json:"exec_err_state" yaml:"exec_err_state"`
	Provenance      Provenance          `json:"provenance,omitempty" yaml:"provenance,omitempty"`
	IsPaused        bool                `json:"is_paused" yaml:"is_paused"`
}
