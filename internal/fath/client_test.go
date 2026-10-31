package fath

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleConfig = `{
  "00000000-0000-0000-0000-000000000000": {
    "sysapName": "SysAP",
    "floorplan": {
      "floors": {
        "01": {
          "name": "Ground Floor",
          "rooms": {
            "0001": { "name": "Living Room" }
          }
        }
      }
    },
    "devices": {
      "ABB700000001": {
        "displayName": "Switch Actuator",
        "channels": {
          "ch0000": {
            "displayName": "Living Room Light",
            "functionID": "0007",
            "floor": "01",
            "room": "0001",
            "inputs": {
              "idp0000": { "pairingID": 1, "value": "0" }
            },
            "outputs": {
              "odp0000": { "pairingID": "256", "value": "1" }
            }
          }
        }
      }
    }
  }
}`

func TestGetConfiguration(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != "installer" || pass != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/fhapi/v1/api/rest/configuration" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleConfig))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	client := NewClient(Options{
		Host:     host,
		Scheme:   "http",
		User:     "installer",
		Password: "secret",
	})

	cfg, err := client.GetConfiguration(context.Background())
	if err != nil {
		t.Fatalf("GetConfiguration: %v", err)
	}
	if len(cfg) != 1 {
		t.Fatalf("expected 1 sysap, got %d", len(cfg))
	}
	for _, sysap := range cfg {
		device, ok := sysap.Devices["ABB700000001"]
		if !ok {
			t.Fatal("expected device ABB700000001")
		}
		channel, ok := device.Channels["ch0000"]
		if !ok {
			t.Fatal("expected channel ch0000")
		}
		if got := channel.Outputs["odp0000"].Value; got != "1" {
			t.Errorf("output value = %q, want %q", got, "1")
		}
		if got := channel.Outputs["odp0000"].PairingID.String(); got != "256" {
			t.Errorf("output pairingID = %q, want %q", got, "256")
		}
		if got := channel.Inputs["idp0000"].PairingID.String(); got != "1" {
			t.Errorf("input pairingID = %q, want %q", got, "1")
		}
		floorName, roomName := sysap.FloorRoomName(channel.Floor, channel.Room)
		if floorName != "Ground Floor" || roomName != "Living Room" {
			t.Errorf("floor/room = %q/%q, want Ground Floor/Living Room", floorName, roomName)
		}
	}
}

func TestSetDatapoint(t *testing.T) {
	var gotMethod, gotPath, gotBody, gotContentType string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"00000000-0000-0000-0000-000000000000": {"result": "OK"}}`))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	client := NewClient(Options{Host: host, Scheme: "http", User: "installer", Password: "secret"})

	err := client.SetDatapoint(context.Background(), "00000000-0000-0000-0000-000000000000", "ABB700000001", "ch0000", "idp0000", "1")
	if err != nil {
		t.Fatalf("SetDatapoint: %v", err)
	}
	if gotMethod != http.MethodPut {
		t.Errorf("method = %q, want PUT", gotMethod)
	}
	wantPath := "/fhapi/v1/api/rest/datapoint/00000000-0000-0000-0000-000000000000/ABB700000001.ch0000.idp0000"
	if gotPath != wantPath {
		t.Errorf("path = %q, want %q", gotPath, wantPath)
	}
	if gotBody != "1" {
		t.Errorf("body = %q, want raw value \"1\" (not JSON)", gotBody)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q", gotContentType)
	}
}

func TestSetDatapointError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "unknown datapoint"}`))
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	client := NewClient(Options{Host: host, Scheme: "http", User: "installer", Password: "secret"})

	if err := client.SetDatapoint(context.Background(), "sysap", "dev", "ch0000", "idp0000", "1"); err == nil {
		t.Fatal("expected error for non-2xx response")
	}
}

func TestGetConfigurationUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	host := strings.TrimPrefix(srv.URL, "http://")
	client := NewClient(Options{Host: host, Scheme: "http", User: "x", Password: "wrong"})

	_, err := client.GetConfiguration(context.Background())
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}
