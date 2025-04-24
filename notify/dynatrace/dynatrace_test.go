package dynatrace

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	commoncfg "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/common/promslog"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/prometheus/alertmanager/config"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/notify/test"
	"github.com/prometheus/alertmanager/types"
)

var receiverConf = `
url: 'https://ab.live.dynatrace.com/api/v2/events/ingest'
entity_selector: 
  type: cloud_app_instance
  label: pod
  expression: some expr
http_config:
  authorization:
    type: Api-Token
    credentials: dt0c01.##.##
`

func TestConstructRelatoK8s(t *testing.T) {

	conf_empty := `
url: 'http://127.0.0.2:5001/'
`

	dynatraceConfig := &config.DynatraceConfig{}
	err := yaml.Unmarshal([]byte(receiverConf), dynatraceConfig)
	if err != nil {
		t.Error(err)
	}

	require.Equal(t, "http://127.0.0.2:5001/", dynatraceConfig.URL.String())
	require.Equal(t, "cloud_app_instance", dynatraceConfig.EntitySelector.Type)
	require.Equal(t, "pod", dynatraceConfig.EntitySelector.Label)
	require.Equal(t, "some expr", dynatraceConfig.EntitySelector.Expression)

	dynatraceConfig = &config.DynatraceConfig{}
	err = yaml.Unmarshal([]byte(conf_empty), dynatraceConfig)
	if err != nil {
		t.Error(err)
	}
	require.Nil(t, dynatraceConfig.EntitySelector)

}

func TestDynatrace(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		dec := json.NewDecoder(r.Body)
		out := make(map[string]interface{})
		err := dec.Decode(&out)
		if err != nil {
			panic(err)
		}
	}))
	defer srv.Close()
	u, _ := url.Parse(srv.URL)

	for _, tc := range []struct {
		title string
		cfg   *config.DynatraceConfig

		retry  bool
		errMsg string
	}{
		{
			title: "full-blown message",
			cfg:   &config.DynatraceConfig{},
			retry: false,
		},
	} {
		t.Run(tc.title, func(t *testing.T) {
			tc.cfg.URL = &config.SecretURL{URL: u}
			tc.cfg.HTTPConfig = &commoncfg.HTTPClientConfig{}
			pd, err := New(tc.cfg, test.CreateTmpl(t), promslog.NewNopLogger())
			require.NoError(t, err)

			ctx := context.Background()
			ctx = notify.WithGroupKey(ctx, "1")

			ok, err := pd.Notify(ctx, []*types.Alert{
				{
					Alert: model.Alert{
						Labels: model.LabelSet{
							"alertname":   "Test Alert Name",
							"lbl1":        "val1",
							"description": "My description",
						},
						StartsAt: time.Now(),
						EndsAt:   time.Now().Add(time.Hour),
					},
				},
			}...)
			if tc.errMsg == "" {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.errMsg)
			}
			require.Equal(t, tc.retry, ok)
		})
	}
}

func TestDynatraceIntegration(t *testing.T) {

	dynatraceConfig := &config.DynatraceConfig{}
	err := yaml.Unmarshal([]byte(receiverConf), dynatraceConfig)
	if err != nil {
		t.Error(err)
	}

	not, err := New(dynatraceConfig, test.CreateTmpl(t), promslog.NewNopLogger())
	if err != nil {
		t.Error(err)
	}

	not.Notify(context.Background(), []*types.Alert{
		{
			Alert: model.Alert{
				Labels: model.LabelSet{
					"alertname":   "Test Alert Name 20",
					"lbl1":        "val2",
					"description": "My description 2",
					"pod": "my-pod-2",
				},
				StartsAt: time.Now(),
				EndsAt:   time.Now().Add(time.Hour),
			},
		},
	}...)

	// tc.cfg.URL = &config.SecretURL{URL: u}
	// tc.cfg.HTTPConfig = &commoncfg.HTTPClientConfig{}
	// pd, err := New(tc.cfg, test.CreateTmpl(t), promslog.NewNopLogger())
	// require.NoError(t, err)

	// ctx := context.Background()
	// ctx = notify.WithGroupKey(ctx, "1")

	// ok, err := pd.Notify(ctx, []*types.Alert{
	// 	{
	// 		Alert: model.Alert{
	// 			Labels: model.LabelSet{
	// 				"alertname":   "Test Alert Name",
	// 				"lbl1":        "val1",
	// 				"description": "My description",
	// 			},
	// 			StartsAt: time.Now(),
	// 			EndsAt:   time.Now().Add(time.Hour),
	// 		},
	// 	},
	// }...)
	// if tc.errMsg == "" {
	// 	require.NoError(t, err)
	// } else {
	// 	require.Error(t, err)
	// 	require.Contains(t, err.Error(), tc.errMsg)
	// }
	// require.Equal(t, tc.retry, ok)
}
