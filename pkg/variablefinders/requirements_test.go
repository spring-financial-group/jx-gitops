package variablefinders_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/variablefinders"
	fakejx "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// generateTestOutput enable to regenerate the expected output
var generateTestOutput = false

func TestFindRequirements(t *testing.T) {
	ns := "jx"
	devGitURL := "https://github.com/myorg/myrepo.git"

	tmpDir := t.TempDir()

	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = devGitURL
	jxClient := fakejx.NewSimpleClientset(devEnv)

	owner := "myorg"
	repo := "somerepo"

	// Test cases for different cloning settings (full, sparse, shallow) and file patterns
	testCasesCloning := []struct {
		cloneType      string
		sparsePatterns []string
	}{
		{
			cloneType:      "full",
			sparsePatterns: nil,
		},
		{
			cloneType:      "full",
			sparsePatterns: []string{"expected.yaml"},
		},
		{
			cloneType:      "sparse",
			sparsePatterns: nil,
		},
		{
			cloneType:      "sparse",
			sparsePatterns: []string{"expected.yaml"},
		},
		{
			cloneType:      "shallow",
			sparsePatterns: nil,
		},
		{
			cloneType:      "shallow",
			sparsePatterns: []string{"expected.yaml"},
		},
	}

	testCases := []struct {
		path        string
		expectError bool
	}{
		{
			path: "disable_env",
		},
		{
			path: "no_settings",
		},
		{
			path: "group_settings",
		},
		{
			path: "group_and_local_settings",
		},
		{
			path: "chart_repo",
		},
		{
			path: "all",
		},
	}

	for _, tc := range testCases {
		for _, cloneCase := range testCasesCloning {
			name := tc.path
			dir := filepath.Join("testdata", name)

			// Setup the fake runner to simulate different clone behaviors
			runner := &fakerunner.FakeRunner{
				CommandRunner: func(command *cmdrunner.Command) (string, error) {
					if command.Name == "git" && len(command.Args) > 1 && command.Args[0] == "clone" {
						if command.Dir == "" {
							return "", errors.Errorf("no dir for git clone")
						}
						devGitPath := filepath.Join(dir, "dev-env")
						destDir := command.Dir
						// Destination directory appended as the final arg
						if len(command.Args) > 2 {
							destDir = command.Args[len(command.Args)-1]
						}

						if cloneCase.cloneType == "sparse" || cloneCase.cloneType == "shallow" {
							err := files.CopyDirOverwrite(devGitPath, destDir)
							if err != nil {
								return "", errors.Wrapf(err, "failed to sparse/shallow clone %s to %s", devGitPath, command.Dir)
							}
						} else {
							err := files.CopyDirOverwrite(devGitPath, destDir)
							if err != nil {
								return "", errors.Wrapf(err, "failed to full clone %s to %s", devGitPath, command.Dir)
							}
						}
						return "", nil
					}
					return "fake " + command.CLI(), nil
				},
			}

			g := cli.NewCLIClient("git", runner.Run)

			sparsePatterns := cloneCase.sparsePatterns

			requirements, err := variablefinders.FindRequirements(g, jxClient, ns, dir, owner, repo, cloneCase.cloneType, sparsePatterns...)

			if tc.expectError {
				require.Error(t, err, "expected error for %s with clone type %s", name, cloneCase.cloneType)
				t.Logf("got expected error %s for %s with clone type %s\n", err.Error(), name, cloneCase.cloneType)
			} else {
				require.NoError(t, err, "should not fail for %s with clone type %s", name, cloneCase.cloneType)
				require.NotNil(t, requirements, "should have got a requirements for %s with clone type %s", name, cloneCase.cloneType)
			}

			expectedPath := filepath.Join(dir, "expected.yml")
			generatedFile := filepath.Join(tmpDir, name+"-"+cloneCase.cloneType+"-requirements.yml")
			err = yamls.SaveFile(requirements, generatedFile)
			require.NoError(t, err, "failed to save generated requirements %s", generatedFile)

			// Generate expected output if necessary
			if generateTestOutput {
				data, err := os.ReadFile(generatedFile)
				require.NoError(t, err, "failed to load %s", generatedFile)

				err = os.WriteFile(expectedPath, data, 0o600)
				require.NoError(t, err, "failed to save file %s", expectedPath)
				continue
			}

			// Compare generated output to expected output
			testhelpers.AssertTextFilesEqual(t, expectedPath, generatedFile, "generated requirements file for test "+name+" with clone type "+cloneCase.cloneType)

			t.Logf("generated file %s is expected for %s with clone type %s\n", generatedFile, name, cloneCase.cloneType)
		}
	}
}
