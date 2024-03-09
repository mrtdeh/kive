package grpc_test

import (
	"reflect"
	"testing"

	"github.com/mrtdeh/kive/pkg/kive"
)

func TestKiveSimple(t *testing.T) {

	if err := kive.Init(nil); err != nil {
		t.Errorf("init error: %v", err)
	}

	t.Run("set-key", func(t *testing.T) {
		res, err := kive.Put("dc1", "testns", "mykey", "123456", 1)
		if err != nil {
			t.Errorf("set error: %v", err)
		}

		if res == nil {
			t.Error("put result is nil")
		}
	})

	t.Run("get-key", func(t *testing.T) {
		val, err := kive.Get("dc1", "testns", "mykey")
		if err != nil {
			t.Errorf("get error: %v", err)
		}

		if val == nil {
			t.Error("val result is nil")
		}

		if kv, ok := val.(kive.KVRequest); ok {
			if kv.Value != "123456" {
				t.Error("val result is not 123456")
			}
		} else {
			t.Error("val result is not kive.KVRequest")
		}

	})

	t.Run("get-namespace", func(t *testing.T) {
		val, err := kive.Get("dc1", "testns", "")
		if err != nil {
			t.Errorf("get error: %v", err)
		}

		if val == nil {
			t.Error("val result is nil")
		}

		if reflect.TypeOf(val).Kind() != reflect.Struct {
			t.Error("val is not Struct")
		}
	})

	t.Run("delete-failed-key", func(t *testing.T) {
		_, err := kive.Del("dc1", "testns", "mykey", 0)
		if err == nil {
			t.Error("delete action must cause error but is nil")
		}
	})

	t.Run("delete-success-key", func(t *testing.T) {
		_, err := kive.Del("dc1", "testns", "mykey", 2)
		if err != nil {
			t.Error("error in delete action : ", err)
		}
	})

}
