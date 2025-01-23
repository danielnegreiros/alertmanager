package dynatrace

import (
	"context"
	"log"
	"log/slog"
	"net/http"

	commoncfg "github.com/prometheus/common/config"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
)

type Notifier struct {
	conf    *config.DynatraceConfig
	tmpl    *template.Template
	logger  *slog.Logger
	client  *http.Client
	retrier *notify.Retrier
}

func New(conf *config.DynatraceConfig, t *template.Template, l *slog.Logger, httpOpts ...commoncfg.HTTPClientOption) (*Notifier, error){
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
	log.Println(n.conf.URL)

	for _, alert := range as {
		log.Println(alert.Name())
		log.Println(alert.Labels)
	}

	return false, nil
}
