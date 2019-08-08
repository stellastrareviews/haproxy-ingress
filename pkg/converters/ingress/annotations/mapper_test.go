/*
Copyright 2019 The HAProxy Ingress Controller Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package annotations

import (
	"reflect"
	"sort"
	"testing"

	hatypes "github.com/jcmoraisjr/haproxy-ingress/pkg/haproxy/types"
)

type ann struct {
	src      *Source
	uri      string
	key      string
	val      string
	expAdded bool
}

var (
	srcing1 = &Source{
		Type:      "ingress",
		Namespace: "default",
		Name:      "ing1",
	}
	srcing2 = &Source{
		Type:      "ingress",
		Namespace: "default",
		Name:      "ing2",
	}
	srcing3 = &Source{
		Type:      "ingress",
		Namespace: "default",
		Name:      "ing3",
	}
	srcing4 = &Source{
		Type:      "ingress",
		Namespace: "default",
		Name:      "ing4",
	}
)

func TestAddAnnotation(t *testing.T) {
	testCases := []struct {
		ann       []ann
		annPrefix string
		getKey    string
		expMiss   bool
		expVal    string
		expLog    string
	}{
		// 0
		{
			ann: []ann{
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/url", "auth-basic", "default/basic2", true},
			},
			annPrefix: "ing/",
			getKey:    "auth-basic",
			expVal:    "default/basic1",
			expLog:    "WARN annotation 'ing/auth-basic' from ingress 'default/ing1' overrides the same annotation with distinct value from [ingress 'default/ing2']",
		},
		// 1
		{
			ann: []ann{
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/url", "auth-basic", "default/basic2", true},
				{srcing3, "/path", "auth-basic", "default/basic3", true},
				{srcing4, "/app", "auth-basic", "default/basic4", true},
			},
			annPrefix: "ing.k8s.io/",
			getKey:    "auth-basic",
			expVal:    "default/basic1",
			expLog:    "WARN annotation 'ing.k8s.io/auth-basic' from ingress 'default/ing1' overrides the same annotation with distinct value from [ingress 'default/ing2' ingress 'default/ing3' ingress 'default/ing4']",
		},
		// 2
		{
			ann: []ann{
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/url", "auth-basic", "default/basic1", true},
				{srcing3, "/path", "auth-basic", "default/basic1", true},
				{srcing4, "/app", "auth-basic", "default/basic2", true},
			},
			annPrefix: "ing.k8s.io/",
			getKey:    "auth-basic",
			expVal:    "default/basic1",
			expLog:    "WARN annotation 'ing.k8s.io/auth-basic' from ingress 'default/ing1' overrides the same annotation with distinct value from [ingress 'default/ing4']",
		},
		// 3
		{
			ann: []ann{
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/", "auth-basic", "default/basic2", false},
			},
			getKey: "auth-basic",
			expVal: "default/basic1",
		},
		// 4
		{
			ann: []ann{
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/url", "auth-basic", "default/basic1", true},
			},
			getKey: "auth-basic",
			expVal: "default/basic1",
		},
		// 5
		{
			ann:     []ann{},
			getKey:  "auth-basic",
			expMiss: true,
		},
	}
	for i, test := range testCases {
		c := setup(t)
		mapper := NewMapBuilder(c.logger, test.annPrefix, map[string]string{}).NewMapper()
		for j, ann := range test.ann {
			if added := mapper.AddAnnotation(ann.src, ann.uri, ann.key, ann.val); added != ann.expAdded {
				t.Errorf("expect added '%t' on '// %d (%d)', but was '%t'", ann.expAdded, i, j, added)
			}
		}
		if _, _, found := mapper.GetStr("error"); found {
			t.Errorf("expect to not find 'error' key on '%d', but was found", i)
		}
		v, _, found := mapper.GetStr(test.getKey)
		if !found {
			if !test.expMiss {
				t.Errorf("expect to find '%s' key on '%d', but was not found", test.getKey, i)
			}
		} else if v != test.expVal {
			t.Errorf("expect '%s' on '%d', but was '%s'", test.expVal, i, v)
		}
		c.logger.CompareLogging(test.expLog)
		c.teardown()
	}
}

func TestGetAnnotation(t *testing.T) {
	testCases := []struct {
		ann       []ann
		annPrefix string
		getKey    string
		expMiss   bool
		expAnnMap []*Map
	}{
		// 0
		{
			ann: []ann{
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/url", "auth-basic", "default/basic2", true},
			},
			getKey: "auth-basic",
			expAnnMap: []*Map{
				{Source: srcing1, URI: "/", Value: "default/basic1"},
				{Source: srcing2, URI: "/url", Value: "default/basic2"},
			},
		},
		// 1
		{
			ann: []ann{
				{srcing1, "/", "auth-type", "basic", true},
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/", "auth-basic", "default/basic2", false},
			},
			getKey: "auth-basic",
			expAnnMap: []*Map{
				{Source: srcing1, URI: "/", Value: "default/basic1"},
			},
		},
		// 2
		{
			ann: []ann{
				{srcing1, "/", "auth-type", "basic", true},
				{srcing1, "/", "auth-basic", "default/basic1", true},
				{srcing2, "/", "auth-basic", "default/basic2", false},
			},
			getKey: "auth-type",
			expAnnMap: []*Map{
				{Source: srcing1, URI: "/", Value: "basic"},
			},
		},
	}
	for i, test := range testCases {
		c := setup(t)
		mapper := NewMapBuilder(c.logger, test.annPrefix, map[string]string{}).NewMapper()
		for j, ann := range test.ann {
			if added := mapper.AddAnnotation(ann.src, ann.uri, ann.key, ann.val); added != ann.expAdded {
				t.Errorf("expect added '%t' on '// %d (%d)', but was '%t'", ann.expAdded, i, j, added)
			}
		}
		annMap, found := mapper.GetStrMap(test.getKey)
		if !found {
			if !test.expMiss {
				t.Errorf("expect to find '%s' key on '%d', but was not found", test.getKey, i)
			}
		} else if !reflect.DeepEqual(annMap, test.expAnnMap) {
			t.Errorf("expected and actual differ on '%d' - expected: %+v - actual: %+v", i, test.expAnnMap, annMap)
		}
		c.teardown()
	}
}

func TestGetDefault(t *testing.T) {
	testCases := []struct {
		annDefaults map[string]string
		ann         map[string]string
		expAnn      map[string]string
	}{
		// 0
		{
			expAnn: map[string]string{
				"timeout-client": "",
			},
		},
		// 1
		{
			annDefaults: map[string]string{
				"timeout-client": "10s",
				"balance":        "roundrobin",
			},
			expAnn: map[string]string{
				"timeout-client": "10s",
				"balance":        "roundrobin",
			},
		},
		// 2
		{
			annDefaults: map[string]string{
				"timeout-client": "10s",
				"balance":        "roundrobin",
			},
			ann: map[string]string{
				"balance": "leastconn",
			},
			expAnn: map[string]string{
				"timeout-client": "10s",
				"balance":        "leastconn",
			},
		},
		// 3
		{
			annDefaults: map[string]string{
				"timeout-client": "10s",
				"balance":        "roundrobin",
			},
			ann: map[string]string{
				"timeout-client": "20s",
			},
			expAnn: map[string]string{
				"timeout-client": "20s",
				"balance":        "roundrobin",
			},
		},
		// 4
		{
			annDefaults: map[string]string{
				"timeout-client": "10s",
				"balance":        "roundrobin",
			},
			ann: map[string]string{
				"timeout-client": "30s",
				"balance":        "leastconn",
			},
			expAnn: map[string]string{
				"timeout-client": "30s",
				"balance":        "leastconn",
			},
		},
	}
	for i, test := range testCases {
		c := setup(t)
		mapper := NewMapBuilder(c.logger, "ing.k8s.io", test.annDefaults).NewMapper()
		mapper.AddAnnotations(&Source{}, "/", test.ann)
		for key, exp := range test.expAnn {
			value := mapper.GetStrValue(key)
			if exp != value {
				t.Errorf("expected key '%s'='%s' on '%d', but was '%s'", key, exp, i, value)
			}
		}
		c.teardown()
	}
}

func TestGetBackendConfig(t *testing.T) {
	testCases := []struct {
		source     Source
		annDefault map[string]string
		keyValues  map[string]map[string]string
		getKeys    []string
		expected   []*BackendConfig
		logging    string
	}{
		// 0
		{
			keyValues: map[string]map[string]string{
				"ann-1": {
					"/": "10",
				},
			},
			getKeys: []string{"ann-1"},
			expected: []*BackendConfig{
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/"}),
					Config: map[string]string{
						"ann-1": "10",
					},
				},
			},
		},
		// 1
		{
			keyValues: map[string]map[string]string{
				"ann-1": {
					"/": "10",
				},
				"ann-2": {
					"/": "10",
				},
			},
			getKeys: []string{"ann-1", "ann-2"},
			expected: []*BackendConfig{
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/"}),
					Config: map[string]string{
						"ann-1": "10",
						"ann-2": "10",
					},
				},
			},
		},
		// 2
		{
			annDefault: map[string]string{
				"ann-1": "0",
			},
			getKeys: []string{"ann-1", "ann-2"},
			keyValues: map[string]map[string]string{
				"ann-1": {
					"/":    "10",
					"/url": "10",
				},
				"ann-2": {
					"/":     "20",
					"/url":  "20",
					"/path": "20",
				},
			},
			expected: []*BackendConfig{
				{
					Paths: hatypes.NewBackendPaths(
						&hatypes.BackendPath{Path: "/"},
						&hatypes.BackendPath{Path: "/url"},
					),
					Config: map[string]string{
						"ann-1": "10",
						"ann-2": "20",
					},
				},
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/path"}),
					Config: map[string]string{
						"ann-1": "0",
						"ann-2": "20",
					},
				},
			},
		},
		// 3
		{
			annDefault: map[string]string{
				"ann-1": "5",
			},
			keyValues: map[string]map[string]string{
				"ann-1": {
					"/url": "10",
				},
			},
			getKeys: []string{"ann-1"},
			expected: []*BackendConfig{
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/url"}),
					Config: map[string]string{
						"ann-1": "10",
					},
				},
			},
		},
		// 4
		{
			annDefault: map[string]string{
				"ann-1": "5",
				"ann-2": "0",
				"ann-3": "0",
			},
			keyValues: map[string]map[string]string{
				"ann-1": {
					"/": "10",
				},
				"ann-2": {
					"/url": "20",
				},
			},
			getKeys: []string{"ann-1", "ann-2", "ann-3"},
			expected: []*BackendConfig{
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/"}),
					Config: map[string]string{
						"ann-1": "10",
						"ann-2": "0",
						"ann-3": "0",
					},
				},
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/url"}),
					Config: map[string]string{
						"ann-1": "5",
						"ann-2": "20",
						"ann-3": "0",
					},
				},
			},
		},
		// 5
		{
			annDefault: map[string]string{
				"ann-1": "0",
			},
			keyValues: map[string]map[string]string{
				"ann-1": {
					"/":    "err",
					"/url": "0",
				},
			},
			getKeys: []string{"ann-1"},
			expected: []*BackendConfig{
				{
					Paths: hatypes.NewBackendPaths(&hatypes.BackendPath{Path: "/"}, &hatypes.BackendPath{Path: "/url"}),
					Config: map[string]string{
						"ann-1": "0",
					},
				},
			},
			source:  Source{Namespace: "default", Name: "ing1", Type: "service"},
			logging: `WARN ignoring invalid int expression on service 'default/ing1': err`,
		},
	}
	validators["ann-1"] = validateInt
	defer delete(validators, "ann-1")
	for i, test := range testCases {
		c := setup(t)
		b := c.createBackendData("default/app", &Source{}, map[string]string{}, test.annDefault)
		for _, kv := range test.keyValues {
			for path := range kv {
				b.backend.AddHostPath("", path)
			}
		}
		for key, values := range test.keyValues {
			for url, value := range values {
				b.mapper.AddAnnotation(&test.source, url, key, value)
			}
		}
		config := b.mapper.GetBackendConfig(b.backend, test.getKeys)
		for _, cfg := range config {
			for i := range cfg.Paths.Items {
				cfg.Paths.Items[i].ID = ""
				cfg.Paths.Items[i].Hostpath = ""
			}
		}
		if !reflect.DeepEqual(config, test.expected) {
			t.Errorf("expected and actual differ on '%d' - expected: %+v - actual: %+v", i, test.expected, config)
		}
		c.logger.CompareLogging(test.logging)
		c.teardown()
	}
}

func TestGetBackendConfigString(t *testing.T) {
	testCases := []struct {
		annDefault map[string]string
		values     map[string]string
		expected   map[string][]string
	}{
		// 0
		{
			values: map[string]string{
				"/":    "20",
				"/url": "30",
			},
			expected: map[string][]string{
				"20": {"/"},
				"30": {"/url"},
			},
		},
		// 1
		{
			values: map[string]string{
				"/":    "20",
				"/url": "20",
			},
			expected: map[string][]string{
				"20": {"/", "/url"},
			},
		},
		// 2
		{
			values: map[string]string{
				"/":     "20",
				"/path": "20",
				"/url":  "10",
			},
			expected: map[string][]string{
				"10": {"/url"},
				"20": {"/", "/path"},
			},
		},
		// 3
		{
			values: map[string]string{
				"/":     "20",
				"/path": "20",
				"/url":  "10",
			},
			expected: map[string][]string{
				"10": {"/url"},
				"20": {"/", "/path"},
			},
		},
	}
	key := "ann-1"
	for i, test := range testCases {
		c := setup(t)
		b := c.createBackendData("default/app", &Source{}, map[string]string{}, test.annDefault)
		for path, value := range test.values {
			b.backend.AddHostPath("", path)
			b.mapper.AddAnnotation(&Source{}, path, key, value)
		}
		config := b.mapper.GetBackendConfigStr(b.backend, key)
		for _, cfg := range config {
			for i := range cfg.Paths.Items {
				cfg.Paths.Items[i].ID = "-"
			}
		}
		sort.SliceStable(config, func(i, j int) bool {
			return config[i].Config < config[j].Config
		})
		expected := []*hatypes.BackendConfigStr{}
		for value, urls := range test.expected {
			paths := hatypes.NewBackendPaths()
			for _, url := range urls {
				paths.Add(&hatypes.BackendPath{
					ID:       "-",
					Hostpath: url,
					Path:     url,
				})
			}
			expected = append(expected, &hatypes.BackendConfigStr{
				Paths:  paths,
				Config: value,
			})
		}
		sort.SliceStable(expected, func(i, j int) bool {
			return expected[i].Config < expected[j].Config
		})
		if !reflect.DeepEqual(config, expected) {
			t.Errorf("expected and actual differ on '%d' - expected: %+v - actual: %+v", i, expected, config)
		}
		c.teardown()
	}
}
