package kafds

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Benny93/kafui/pkg/api"
	"github.com/birdayz/kaf/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// withRegistry points the active cluster at the given base URL(s) for the
// duration of fn, restoring the previous cluster afterwards.
func withRegistry(t *testing.T, url string, creds *config.SchemaRegistryCredentials) func() {
	t.Helper()
	prev := currentCluster
	currentCluster = &config.Cluster{SchemaRegistryURL: url, SchemaRegistryCredentials: creds}
	return func() { currentCluster = prev }
}

func TestRegistryClient_Verbs_AuthAndContentType(t *testing.T) {
	var gotMethod, gotAuth, gotAccept, gotContentType, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		gotContentType = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()
	defer withRegistry(t, srv.URL, &config.SchemaRegistryCredentials{Username: "u", Password: "p"})()

	kp := KafkaDataSourceKaf{}
	rc, err := kp.newRegistryClient()
	require.NoError(t, err)
	require.NotNil(t, rc)

	t.Run("get sends auth + accept", func(t *testing.T) {
		var out map[string]bool
		require.NoError(t, rc.doGet("/x", &out))
		assert.Equal(t, http.MethodGet, gotMethod)
		assert.True(t, strings.HasPrefix(gotAuth, "Basic "))
		assert.Equal(t, registryContentType, gotAccept)
		assert.True(t, out["ok"])
	})

	t.Run("post sends body + content-type", func(t *testing.T) {
		require.NoError(t, rc.doPost("/x", map[string]string{"schema": "s"}, nil))
		assert.Equal(t, http.MethodPost, gotMethod)
		assert.Equal(t, registryContentType, gotContentType)
		assert.JSONEq(t, `{"schema":"s"}`, gotBody)
	})

	t.Run("put and delete verbs", func(t *testing.T) {
		require.NoError(t, rc.doPut("/x", map[string]string{"a": "b"}, nil))
		assert.Equal(t, http.MethodPut, gotMethod)
		require.NoError(t, rc.doDelete("/x", nil))
		assert.Equal(t, http.MethodDelete, gotMethod)
	})
}

// registryErrorHandler serves a fixed status + registry error body.
func registryErrorHandler(status, errorCode int, message string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"error_code": errorCode, "message": message})
	}
}

func TestMapRegistryError(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		code    int
		message string
		check   func(t *testing.T, err error)
	}{
		{"subject not found by 404", http.StatusNotFound, 40401, "Subject not found", func(t *testing.T, err error) {
			var e api.SubjectNotFoundError
			assert.True(t, errors.As(err, &e))
			assert.Equal(t, "orders", e.Subject)
		}},
		{"version not found by 40402", http.StatusNotFound, 40402, "Version not found", func(t *testing.T, err error) {
			var e api.SchemaVersionNotFoundError
			assert.True(t, errors.As(err, &e))
		}},
		{"incompatible by 409", http.StatusConflict, 0, "reader incompatible", func(t *testing.T, err error) {
			var e api.SchemaIncompatibleError
			require.True(t, errors.As(err, &e))
			assert.Equal(t, "reader incompatible", e.Message)
		}},
		{"invalid by 422", http.StatusUnprocessableEntity, 42201, "Invalid schema", func(t *testing.T, err error) {
			var e api.SchemaValidationError
			require.True(t, errors.As(err, &e))
			assert.Equal(t, "Invalid schema", e.Message)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(registryErrorHandler(tc.status, tc.code, tc.message))
			defer srv.Close()
			defer withRegistry(t, srv.URL, nil)()

			kp := KafkaDataSourceKaf{}
			rc, _ := kp.newRegistryClient()
			err := rc.doGet("/subjects/orders/versions/latest", nil)
			tc.check(t, mapRegistryError(err, "orders", 3))
		})
	}
}

func TestGetSchemaVersions(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/subjects/orders/versions":
			w.Write([]byte(`[1,2]`))
		case strings.HasSuffix(r.URL.Path, "/versions/1"):
			w.Write([]byte(`{"version":1,"id":10,"schemaType":"AVRO"}`))
		case strings.HasSuffix(r.URL.Path, "/versions/2"):
			w.Write([]byte(`{"version":2,"id":11}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error_code": 40401, "message": "not found"})
		}
	}))
	defer srv.Close()
	defer withRegistry(t, srv.URL, nil)()

	kp := KafkaDataSourceKaf{}
	versions, err := kp.GetSchemaVersions("orders")
	require.NoError(t, err)
	require.Len(t, versions, 2)
	assert.Equal(t, 1, versions[0].Version)
	assert.Equal(t, 2, versions[1].Version)
	assert.Equal(t, "AVRO", versions[1].SchemaType) // empty defaults to AVRO

	t.Run("unknown subject", func(t *testing.T) {
		_, err := kp.GetSchemaVersions("nope")
		var e api.SubjectNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}

func TestCompatibility_GetGlobalAndSubject(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/config":
			w.Write([]byte(`{"compatibilityLevel":"BACKWARD"}`))
		case "/config/orders-value":
			w.Write([]byte(`{"compatibilityLevel":"FULL"}`))
		case "/config/no-config-value":
			w.WriteHeader(http.StatusNotFound)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{"error_code": 40408, "message": "Subject not configured"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	defer withRegistry(t, srv.URL, nil)()

	kp := KafkaDataSourceKaf{}

	global, err := kp.GetGlobalCompatibility()
	require.NoError(t, err)
	assert.Equal(t, api.CompatibilityBackward, global)

	level, specific, err := kp.GetSubjectCompatibility("orders-value")
	require.NoError(t, err)
	assert.True(t, specific)
	assert.Equal(t, api.CompatibilityFull, level)

	t.Run("falls back to global", func(t *testing.T) {
		level, specific, err := kp.GetSubjectCompatibility("no-config-value")
		require.NoError(t, err)
		assert.False(t, specific)
		assert.Equal(t, api.CompatibilityBackward, level)
	})
}

func TestCompatibility_SetGlobalAndSubject(t *testing.T) {
	var gotPath, gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	defer withRegistry(t, srv.URL, nil)()

	kp := KafkaDataSourceKaf{}

	require.NoError(t, kp.SetGlobalCompatibility(api.CompatibilityForward))
	assert.Equal(t, http.MethodPut, gotMethod)
	assert.Equal(t, "/config", gotPath)
	assert.JSONEq(t, `{"compatibility":"FORWARD"}`, gotBody)

	require.NoError(t, kp.SetSubjectCompatibility("orders-value", api.CompatibilityFull))
	assert.Equal(t, "/config/orders-value", gotPath)
	assert.JSONEq(t, `{"compatibility":"FULL"}`, gotBody)

	t.Run("invalid enum rejected without HTTP call", func(t *testing.T) {
		called := false
		srv2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true }))
		defer srv2.Close()
		defer withRegistry(t, srv2.URL, nil)()
		err := kp.SetGlobalCompatibility(api.CompatibilityLevel("NOPE"))
		require.Error(t, err)
		assert.False(t, called)
	})
}

func TestRegisterSchema(t *testing.T) {
	t.Run("success returns id and version", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				w.Write([]byte(`{"id":42}`))
				return
			}
			// versions/latest re-fetch
			w.Write([]byte(`{"subject":"orders-value","version":4,"id":42,"schemaType":"AVRO"}`))
		}))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()

		kp := KafkaDataSourceKaf{}
		schema, err := kp.RegisterSchema("orders-value", `{"type":"string"}`, "")
		require.NoError(t, err)
		assert.Equal(t, 42, schema.ID)
		assert.Equal(t, 4, schema.Version)
	})

	t.Run("409 maps to incompatible", func(t *testing.T) {
		srv := httptest.NewServer(registryErrorHandler(http.StatusConflict, 0, "incompatible with v3"))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		_, err := kp.RegisterSchema("orders-value", "{}", "")
		var e api.SchemaIncompatibleError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, "incompatible with v3", e.Message)
	})

	t.Run("422 maps to validation", func(t *testing.T) {
		srv := httptest.NewServer(registryErrorHandler(http.StatusUnprocessableEntity, 42201, "bad avro"))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		_, err := kp.RegisterSchema("orders-value", "{", "")
		var e api.SchemaValidationError
		require.True(t, errors.As(err, &e))
		assert.Equal(t, "bad avro", e.Message)
	})
}

func TestCheckSchemaCompatibility(t *testing.T) {
	t.Run("compatible", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Contains(t, r.URL.RawQuery, "verbose=true")
			w.Write([]byte(`{"is_compatible":true}`))
		}))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		ok, msgs, err := kp.CheckSchemaCompatibility("orders-value", "{}", "")
		require.NoError(t, err)
		assert.True(t, ok)
		assert.Empty(t, msgs)
	})

	t.Run("incompatible with messages", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"is_compatible":false,"messages":["field removed"]}`))
		}))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		ok, msgs, err := kp.CheckSchemaCompatibility("orders-value", "{}", "")
		require.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, []string{"field removed"}, msgs)
	})

	t.Run("unknown subject", func(t *testing.T) {
		srv := httptest.NewServer(registryErrorHandler(http.StatusNotFound, 40401, "Subject not found"))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		_, _, err := kp.CheckSchemaCompatibility("nope", "{}", "")
		var e api.SubjectNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}

func TestDeleteSubjectAndVersion(t *testing.T) {
	t.Run("soft delete subject", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Write([]byte(`[1,2,3]`))
		}))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		deleted, err := kp.DeleteSubject("orders-value", false)
		require.NoError(t, err)
		assert.Equal(t, []int{1, 2, 3}, deleted)
		assert.Empty(t, gotQuery, "soft delete must not set permanent=true")
	})

	t.Run("hard delete subject sets permanent", func(t *testing.T) {
		var gotQuery string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotQuery = r.URL.RawQuery
			w.Write([]byte(`[1]`))
		}))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		_, err := kp.DeleteSubject("orders-value", true)
		require.NoError(t, err)
		assert.Equal(t, "permanent=true", gotQuery)
	})

	t.Run("delete latest version via sentinel", func(t *testing.T) {
		var gotPath string
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			gotPath = r.URL.Path
			w.Write([]byte(`3`))
		}))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		require.NoError(t, kp.DeleteSchemaVersion("orders-value", -1, false))
		assert.Equal(t, "/subjects/orders-value/versions/latest", gotPath)
	})

	t.Run("missing version maps to version not found", func(t *testing.T) {
		srv := httptest.NewServer(registryErrorHandler(http.StatusNotFound, 40402, "Version not found"))
		defer srv.Close()
		defer withRegistry(t, srv.URL, nil)()
		kp := KafkaDataSourceKaf{}
		err := kp.DeleteSchemaVersion("orders-value", 99, false)
		var e api.SchemaVersionNotFoundError
		assert.True(t, errors.As(err, &e))
	})
}

func TestNotConfigured(t *testing.T) {
	defer withRegistry(t, "", nil)()
	kp := KafkaDataSourceKaf{}

	// Listing returns empty, not an error.
	schemas, err := kp.GetSchemas()
	require.NoError(t, err)
	assert.Empty(t, schemas)

	// Content/mutation error with the typed not-configured error.
	_, err = kp.GetSchemaContent("orders", 0)
	var e api.SchemaRegistryNotConfiguredError
	assert.True(t, errors.As(err, &e))
	_, err = kp.GetSchemaVersions("orders")
	assert.True(t, errors.As(err, &e))
}

func TestFailover(t *testing.T) {
	t.Run("first URL dead, second serves", func(t *testing.T) {
		dead := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		deadURL := dead.URL
		dead.Close() // now unreachable

		live := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`["a","b"]`))
		}))
		defer live.Close()
		defer withRegistry(t, deadURL+","+live.URL, nil)()

		kp := KafkaDataSourceKaf{}
		schemas, err := kp.GetSchemas()
		require.NoError(t, err)
		assert.Len(t, schemas, 2)
	})

	t.Run("all dead returns no-live error", func(t *testing.T) {
		s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		u1, u2 := s1.URL, s2.URL
		s1.Close()
		s2.Close()
		defer withRegistry(t, u1+","+u2, nil)()

		kp := KafkaDataSourceKaf{}
		_, err := kp.GetSchemas()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no live schema registry instances")
	})

	t.Run("HTTP 4xx does not trigger failover", func(t *testing.T) {
		var hits int
		bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits++
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"message":"boom"}`))
		}))
		defer bad.Close()
		var backupHits int
		backup := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			backupHits++
			w.Write([]byte(`[]`))
		}))
		defer backup.Close()
		defer withRegistry(t, bad.URL+","+backup.URL, nil)()

		kp := KafkaDataSourceKaf{}
		_, err := kp.GetSchemas()
		require.Error(t, err)
		assert.Equal(t, 1, hits)
		assert.Equal(t, 0, backupHits, "an HTTP status must not fail over to the next URL")
	})
}
