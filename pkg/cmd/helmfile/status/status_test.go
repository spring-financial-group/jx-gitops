package status_test

import (
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/status"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"testing"
	"time"
)

const (
	prod    = "prod"
	preProd = "pre-prod"
	staging = "staging"
	preview = "preview"

	repoOwner = "fakeOwner"
	repoName  = "fakeRepo"
)

var fullRepoName = filepath.Join(repoOwner, repoName)

func TestHemlfileStatus(t *testing.T) {
	testCases := []struct {
		name                string
		existingDeployments []*scm.Deployment
		expectedDeployments []*scm.Deployment
		expectedStatuses    map[string][]*scm.DeploymentStatus
	}{
		{
			name: "Existing deployment with same ref (no new status)",
			existingDeployments: []*scm.Deployment{
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Production",
					OriginalEnvironment:   "Production",
					ProductionEnvironment: true,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-1",
					Payload:               "",
				},
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Staging",
					OriginalEnvironment:   "Staging",
					ProductionEnvironment: false,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-2",
					Payload:               "",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Production",
					OriginalEnvironment:   "Production",
					ProductionEnvironment: true,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-1",
					Payload:               "",
				},
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Staging",
					OriginalEnvironment:   "Staging",
					ProductionEnvironment: false,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-2",
					Payload:               "",
				},
			},
			expectedStatuses: map[string][]*scm.DeploymentStatus{},
		},
		{
			name: "Existing deployment with different ref (status created)",
			existingDeployments: []*scm.Deployment{
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Production",
					OriginalEnvironment:   "Production",
					ProductionEnvironment: true,
					Description:           "release nodey560 for version 0.0.1",
					Task:                  "deploy",
					Ref:                   "v0.0.1",
					ID:                    "deployment-1",
					Payload:               "",
				},
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Staging",
					OriginalEnvironment:   "Staging",
					ProductionEnvironment: false,
					Description:           "release nodey560 for version 0.0.1",
					Task:                  "deploy",
					Ref:                   "v0.0.1",
					ID:                    "deployment-2",
					Payload:               "",
				},
			},
			expectedDeployments: []*scm.Deployment{
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Production",
					OriginalEnvironment:   "Production",
					ProductionEnvironment: true,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-1",
					Payload:               "",
				},
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Staging",
					OriginalEnvironment:   "Staging",
					ProductionEnvironment: false,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-2",
					Payload:               "",
				},
			},
			expectedStatuses: map[string][]*scm.DeploymentStatus{
				fullRepoName + "/deployment-1": {
					{
						ID:              "status-1",
						State:           "success",
						Description:     "Deployment 0.0.2",
						Environment:     "Production",
						EnvironmentLink: "https://github.com/jstrachan/jx-demo-gke2-dev.git",
					},
				},
				fullRepoName + "/deployment-2": {
					{
						ID:              "status-1",
						State:           "success",
						Description:     "Deployment 0.0.2",
						Environment:     "Staging",
						EnvironmentLink: "https://github.com/jstrachan/jx-demo-gke2-dev.git",
					},
				},
			},
		},
		{
			name:                "No existing deployments (new deployments & status)",
			existingDeployments: []*scm.Deployment{},
			expectedDeployments: []*scm.Deployment{
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Production",
					OriginalEnvironment:   "Production",
					ProductionEnvironment: true,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-1",
					Payload:               "",
				},
				{
					Name:                  repoName,
					Namespace:             "jstrachan",
					Environment:           "Staging",
					OriginalEnvironment:   "Staging",
					ProductionEnvironment: false,
					Description:           "release nodey560 for version 0.0.2",
					Task:                  "deploy",
					Ref:                   "v0.0.2",
					ID:                    "deployment-2",
					Payload:               "",
				},
			},
			expectedStatuses: map[string][]*scm.DeploymentStatus{
				fullRepoName + "/deployment-1": {
					{
						ID:              "status-1",
						State:           "success",
						Description:     "Deployment 0.0.2",
						Environment:     "Production",
						EnvironmentLink: "https://github.com/jstrachan/jx-demo-gke2-dev.git",
					},
				},
				fullRepoName + "/deployment-2": {
					{
						ID:              "status-1",
						State:           "success",
						Description:     "Deployment 0.0.2",
						Environment:     "Staging",
						EnvironmentLink: "https://github.com/jstrachan/jx-demo-gke2-dev.git",
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, o := status.NewCmdHelmfileStatus()
			o.Dir = "testdata"
			o.TestGitToken = "faketoken"
			o.DeployOffset = ""
			cutoff, err := time.Parse(time.RFC3339, "2023-01-25T08:38:47Z")
			require.NoError(t, err, "failed to parse time")
			o.DeployCutoff = cutoff

			var data *fake.Data
			scmClient, data := fake.NewDefault()
			data.Deployments = map[string][]*scm.Deployment{fullRepoName: tc.existingDeployments}

			o.SCMClients = map[string]*scm.Client{
				"https://fake.com": scmClient,
			}

			err = o.Run()
			require.NoError(t, err, "failed to run")
			assert.Equal(t, tc.expectedStatuses, data.DeploymentStatus)
			assert.Equal(t, tc.expectedDeployments, data.Deployments[fullRepoName])
		})
	}
}

//func TestNewCmdHelmfileStatus_FindExistingDeployment(t *testing.T) {
//	fakeDeployments := []*scm.Deployment{
//		{
//			Name:        repoName,
//			Environment: prod,
//		},
//		{
//			Name:        repoName,
//			Environment: preProd,
//		},
//		{
//			Name:        repoName,
//			Environment: staging,
//		},
//	}
//
//	type inputArgs struct {
//		fullRepoName string
//		environment  string
//	}
//
//	testCases := []struct {
//		name               string
//		testArgs           inputArgs
//		currentDeployments []*scm.Deployment
//		expectedDeployment *scm.Deployment
//	}{
//		{
//			name: "correct name and env",
//			testArgs: inputArgs{
//				fullRepoName: fullRepoName,
//				environment:  prod,
//			},
//			expectedDeployment: &scm.Deployment{
//				Name:        repoName,
//				Environment: prod,
//			},
//		},
//		{
//			name: "no existing deployment for name",
//			testArgs: inputArgs{
//				fullRepoName: repoOwner + "/reallyFakeRepo",
//				environment:  prod,
//			},
//			expectedDeployment: nil,
//		},
//		{
//			name: "no existing deployment for env",
//			testArgs: inputArgs{
//				fullRepoName: fullRepoName,
//				environment:  preview,
//			},
//			expectedDeployment: nil,
//		},
//	}
//
//	testOpts := status.Options{}
//
//	var data *fake.Data
//	testOpts.ScmClient, data = fake.NewDefault()
//	data.Deployments = map[string][]*scm.Deployment{fullRepoName: fakeDeployments}
//
//	for _, tt := range testCases {
//		t.Run(tt.name, func(t *testing.T) {
//			actualDeployment, err := testOpts.FindExistingDeploymentInEnvironment(context.TODO(), tt.testArgs.fullRepoName, tt.testArgs.environment)
//			assert.NoError(t, err)
//			assert.Equal(t, tt.expectedDeployment, actualDeployment)
//		})
//	}
//}
