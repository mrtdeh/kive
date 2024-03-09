package integration_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestClusterNodes(t *testing.T) {

	tests := map[string]struct {
		addr     string
		expected expectation
	}{
		"server:9991": {
			addr: "http://localhost:9991/nodes",
			expected: expectation{
				count: 23,
				err:   nil,
			},
		},
		"server:9992": {
			addr: "http://localhost:9992/nodes",
			expected: expectation{
				count: 6,
				err:   nil,
			},
		},
		"server:9993": {
			addr: "http://localhost:9993/nodes",
			expected: expectation{
				count: 12,
				err:   nil,
			},
		},
		"server:9994": {
			addr: "http://localhost:9994/nodes",
			expected: expectation{
				count: 6,
				err:   nil,
			},
		},
	}

	for scenario, tt := range tests {
		func(addr string, e expectation) {
			t.Run(scenario, func(t *testing.T) {
				t.Parallel()

				res, err := http.Get(addr)
				if err != nil {
					t.Error("error in get request : ", err)
				}
				defer res.Body.Close()

				body, _ := io.ReadAll(res.Body)

				var m map[string]interface{}
				err = json.Unmarshal(body, &m)
				if err != nil {
					t.Error("error in unmarshal : ", err)
				}

				nodes := m["result"].([]interface{})
				if len(nodes) != e.count {
					t.Errorf("expected %d, got %d", e.count, len(nodes))
				}

			})
		}(tt.addr, tt.expected)
	}
}
