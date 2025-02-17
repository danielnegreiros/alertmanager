package dynatrace

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	commoncfg "github.com/prometheus/common/config"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
)

func getEntitySelectorExpr(type_ string, value string) string {
	switch type_ {
	case "cloud_app_instance":
		return fmt.Sprintf("type(CLOUD_APPLICATION_INSTANCE),entityName.contains(%s)", value)
	case "custom_expression":
		return value
	default:
		return ""
	}
}

// dynatraceEvent is a struct Dynatrace Problem Event representation
type dynatraceEvent struct {
	EventType        string            `json:"eventType"`
	Title            string            `json:"title"`
	Source           string            `json:"source"`
	Description      string            `json:"description"`
	Timeout          string            `json:"timeout"`
	CustomProperties map[string]string `json:"properties"`
	EntitySelector   string            `json:"entitySelector,omitempty"`
}

func newDynatraceEvent(alert *types.Alert) *dynatraceEvent {

	customProperties := make(map[string]string)
	for key, value := range alert.Labels {
		customProperties[string(key)] = string(value)
	}

	for key, value := range alert.Annotations {
		customProperties[string(key)] = string(value)
	}

	return &dynatraceEvent{
		EventType:        "CUSTOM_ALERT",
		Title:            customProperties["alertname"],
		Source:           "Prometheus Alertmanager",
		Description:      customProperties["description"],
		Timeout:          "15",
		CustomProperties: customProperties,
	}
}

type Notifier struct {
	conf    *config.DynatraceConfig
	tmpl    *template.Template
	logger  *slog.Logger
	client  *http.Client
	retrier *notify.Retrier
}

func New(conf *config.DynatraceConfig, t *template.Template, l *slog.Logger, httpOpts ...commoncfg.HTTPClientOption) (*Notifier, error) {
	l.Info("set up Dynatrace receiver", "endpoint", conf.URL.String())
	client, err := commoncfg.NewClientFromConfig(*conf.HTTPConfig, "dynatrace", httpOpts...)
	if err != nil {
		return nil, err
	}
	return &Notifier{
		conf:   conf,
		tmpl:   t,
		logger: l,
		client: client,
		// Webhooks are assumed to respond with 2xx response codes on a successful
		// request and 5xx response codes are assumed to be recoverable.
		retrier: &notify.Retrier{},
	}, nil
}

func (n *Notifier) Notify(ctx context.Context, as ...*types.Alert) (bool, error) {

	for _, alert := range as {
		event := newDynatraceEvent(alert)
		if n.conf.EntitySelector != nil {
			event.EntitySelector = getEntitySelectorExpr(n.conf.EntitySelector.Type, event.CustomProperties[n.conf.EntitySelector.Label])
		}

		buf := &bytes.Buffer{}
		err := json.NewEncoder(buf).Encode(event)
		if err != nil {
			return false, fmt.Errorf("not possible to encode alert")
		}

		n.logger.Info(n.conf.URL.String())
		resp, err := notify.PostJSON(context.TODO(), n.client, n.conf.URL.String(), buf)
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusCreated {
			return false, fmt.Errorf("not possible to senf alert")
		}

		if err != nil {
			return false, fmt.Errorf("not possible to senf alert")
		}

		n.logger.Error("alert posted with success", "title", event.Title)

	}

	return false, nil
}
