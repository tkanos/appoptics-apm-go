package ao_test

import (
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/appoptics/appoptics-apm-go/v1/ao/internal/config"
	g "github.com/appoptics/appoptics-apm-go/v1/ao/internal/graphtest"
	"github.com/appoptics/appoptics-apm-go/v1/ao/internal/reporter"
	"github.com/stretchr/testify/assert"
)

func TestTriggerTrace(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;pd_keys=lo:se,check-id:123",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "test.com", n.Map["HTTP-Host"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
		}},
	})
	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=ok", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "01"))
}

func TestTriggerTraceNoSetting(t *testing.T) {

	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.NoSettingST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;pd_keys=lo:se,check-id:123",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(0)
	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=settings-not-available", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "00"))
}

func TestTriggerTraceWithCustomKey(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.TriggerTraceOnlyST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;custom_key1=value1",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "value1", n.Map["custom_key1"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
		}},
	})
	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=ok", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "01"))
}

func TestTriggerTraceRateLimited(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.LimitedTriggerTraceST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;custom_key1=value1",
	}

	var rrs []*httptest.ResponseRecorder
	numRequests := 5
	for i := 0; i < numRequests; i++ {
		rrs = append(rrs, httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd))
	}

	r.Close(0) // Don't check number of events here

	numEvts := len(r.EventBufs)
	assert.True(t, numEvts < 10)

	limited := 0
	triggerTraced := 0

	for _, rr := range rrs {
		rsp := rr.Header().Get("X-Trace-Options-Response")
		if rsp == "trigger_trace=ok" {
			triggerTraced++
		} else if rsp == "trigger_trace=rate-exceeded" {
			limited++
		}
	}
	assert.True(t, (limited+triggerTraced) == numRequests)
	assert.True(t, triggerTraced*2 == numEvts,
		fmt.Sprintf("triggerTraced=%d, numEvts=%d", triggerTraced, numEvts))
}

func TestTriggerTraceDisabled(t *testing.T) {
	_ = os.Setenv("APPOPTICS_TRIGGER_TRACE", "false")
	_ = config.Load()

	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;pd_keys=lo:se,check-id:123",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(0)

	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=disabled", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "00"))

	_ = os.Unsetenv("APPOPTICS_TRIGGER_TRACE")
	_ = config.Load()
}

func TestTriggerTraceEnabledTracingModeDisabled(t *testing.T) {
	_ = os.Setenv("APPOPTICS_TRACING_MODE", "disabled")
	_ = config.Load()

	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;pd_keys=lo:se,check-id:123",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(0)

	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=tracing-disabled", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "00"))

	_ = os.Unsetenv("APPOPTICS_TRACING_MODE")
	_ = config.Load()
}

func TestNoTriggerTrace(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "pd_keys=lo:se,check-id:123;custom_key1=value1",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "test.com", n.Map["HTTP-Host"])
			assert.Equal(t, "value1", n.Map["custom_key1"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
		}},
	})
	rHeader := rr.Header()
	assert.Empty(t, rHeader.Get("X-Trace-Options-Response"))
	assert.NotEmpty(t, rHeader.Get("X-Trace"))
}

func TestNoTriggerTraceInvalidFlag(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace=1;tigger_trace",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "test.com", n.Map["HTTP-Host"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
		}},
	})
	rHeader := rr.Header()
	assert.EqualValues(t, "ignored=tigger_trace,trigger_trace", rHeader.Get("X-Trace-Options-Response"))
	assert.NotEmpty(t, rHeader.Get("X-Trace"))
}

func TestTriggerTraceInvalidFlag(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace;foo=bar;app_id=123",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(2)
	g.AssertGraph(t, r.EventBufs, 2, g.AssertNodeMap{
		// entry event should have no edges
		{"http.HandlerFunc", "entry"}: {Edges: g.Edges{}, Callback: func(n g.Node) {
			assert.Equal(t, "test.com", n.Map["HTTP-Host"])
		}},
		{"http.HandlerFunc", "exit"}: {Edges: g.Edges{{"http.HandlerFunc", "entry"}}, Callback: func(n g.Node) {
		}},
	})
	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=ok;ignored=foo,app_id", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "01"))
}

func TestTriggerTraceXTraceBothValid(t *testing.T) {
	r := reporter.SetTestReporter(reporter.TestReporterSettingType(reporter.DefaultST))
	hd := map[string]string{
		"X-Trace-Options": "trigger_trace",
		"X-Trace":         "2B987445277543FF9C151D0CDE6D29B6E21603D5DB2C5EFEA7749039AF00",
	}

	rr := httpTestWithEndpointWithHeaders(handler200, "http://test.com/hello", hd)
	r.Close(0)
	rHeader := rr.Header()
	assert.EqualValues(t, "trigger_trace=ignored", rHeader.Get("X-Trace-Options-Response"))
	assert.True(t, strings.HasSuffix(rHeader.Get("X-Trace"), "00"))
}
