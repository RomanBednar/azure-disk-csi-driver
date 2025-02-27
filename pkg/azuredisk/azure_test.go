/*
Copyright 2020 The Kubernetes Authors.

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

package azuredisk

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func skipIfTestingOnWindows(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping tests on Windows")
	}
}

func TestGetCloudProvider(t *testing.T) {
	// skip for now as this is very flaky on Windows
	skipIfTestingOnWindows(t)
	fakeCredFile := "fake-cred-file.json"
	fakeKubeConfig := "fake-kube-config"
	emptyKubeConfig := "empty-kube-config"
	fakeContent := `
apiVersion: v1
clusters:
- cluster:
    server: https://localhost:8080
  name: foo-cluster
contexts:
- context:
    cluster: foo-cluster
    user: foo-user
    namespace: bar
  name: foo-context
current-context: foo-context
kind: Config
users:
- name: foo-user
  user:
    exec:
      apiVersion: client.authentication.k8s.io/v1alpha1
      args:
      - arg-1
      - arg-2
      command: foo-command
`

	err := createTestFile(emptyKubeConfig)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		if err := os.Remove(emptyKubeConfig); err != nil {
			t.Error(err)
		}
	}()

	tests := []struct {
		desc        string
		kubeconfig  string
		expectedErr error
	}{
		{
			desc:        "[failure] out of cluster, no kubeconfig, no credential file",
			kubeconfig:  "",
			expectedErr: fmt.Errorf("load azure config from file(%s) failed with open %s: no such file or directory", DefaultCredFilePathLinux, DefaultCredFilePathLinux),
		},
		{
			desc:        "[failure] out of cluster & in cluster, specify a non-exist kubeconfig, no credential file",
			kubeconfig:  "/tmp/non-exist.json",
			expectedErr: fmt.Errorf("load azure config from file(%s) failed with open %s: no such file or directory", DefaultCredFilePathLinux, DefaultCredFilePathLinux),
		},
		{
			desc:        "[failure] out of cluster & in cluster, specify a empty kubeconfig, no credential file",
			kubeconfig:  emptyKubeConfig,
			expectedErr: fmt.Errorf("failed to get KubeClient: invalid configuration: no configuration has been provided, try setting KUBERNETES_MASTER environment variable"),
		},
		{
			desc:        "[failure] out of cluster & in cluster, specify a fake kubeconfig, no credential file",
			kubeconfig:  fakeKubeConfig,
			expectedErr: fmt.Errorf("load azure config from file(%s) failed with open %s: no such file or directory", DefaultCredFilePathLinux, DefaultCredFilePathLinux),
		},
		{
			desc:        "[success] out of cluster & in cluster, no kubeconfig, a fake credential file",
			kubeconfig:  "",
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		if test.desc == "[failure] out of cluster & in cluster, specify a fake kubeconfig, no credential file" {
			err := createTestFile(fakeKubeConfig)
			if err != nil {
				t.Error(err)
			}
			defer func() {
				if err := os.Remove(fakeKubeConfig); err != nil {
					t.Error(err)
				}
			}()

			if err := ioutil.WriteFile(fakeKubeConfig, []byte(fakeContent), 0666); err != nil {
				t.Error(err)
			}
		}
		if test.desc == "[success] out of cluster & in cluster, no kubeconfig, a fake credential file" {
			err := createTestFile(fakeCredFile)
			if err != nil {
				t.Error(err)
			}
			defer func() {
				if err := os.Remove(fakeCredFile); err != nil {
					t.Error(err)
				}
			}()

			originalCredFile, ok := os.LookupEnv(DefaultAzureCredentialFileEnv)
			if ok {
				defer os.Setenv(DefaultAzureCredentialFileEnv, originalCredFile)
			} else {
				defer os.Unsetenv(DefaultAzureCredentialFileEnv)
			}
			os.Setenv(DefaultAzureCredentialFileEnv, fakeCredFile)
		}
		_, err := GetCloudProvider(test.kubeconfig, "", "")
		if !reflect.DeepEqual(err, test.expectedErr) {
			t.Errorf("desc: %s,\n input: %q, GetCloudProvider err: %v, expectedErr: %v", test.desc, test.kubeconfig, err, test.expectedErr)
		}
	}
}

func createTestFile(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return nil
}

func TestIsAzureStackCloud(t *testing.T) {
	tests := []struct {
		cloud                  string
		disableAzureStackCloud bool
		expectedResult         bool
	}{
		{
			cloud:                  "AzurePublicCloud",
			disableAzureStackCloud: false,
			expectedResult:         false,
		},
		{
			cloud:                  "",
			disableAzureStackCloud: true,
			expectedResult:         false,
		},
		{
			cloud:                  azureStackCloud,
			disableAzureStackCloud: false,
			expectedResult:         true,
		},
		{
			cloud:                  azureStackCloud,
			disableAzureStackCloud: true,
			expectedResult:         false,
		},
	}

	for i, test := range tests {
		result := IsAzureStackCloud(test.cloud, test.disableAzureStackCloud)
		assert.Equal(t, test.expectedResult, result, "TestCase[%d]", i)
	}
}
