package mirror

import (
	"fmt"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/ghpages"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/chartmuseum/helm-push/pkg/chartmuseum"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/httphelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Escapes any {{ or }} characters in the YAML files so they can be included in a helm chart
`)

	cmdExample = templates.Examples(`
		# escapes any yaml files so they can be included in a helm chart 
		%s helm escape --dir myyaml
	`)
)

// Options the options for the command
type Options struct {
	scmhelpers.Factory
	Dir              string
	RepositoriesFile string
	Branch           string
	RepoURL          string
	CommitMessage    string
	Excludes         []string
	NoPush           bool
	GitClient        gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
	RepoType         string // "Git" or "ChartMuseum"
}

// NewCmdMirror creates a command object for the command
func NewCmdMirror() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "mirror",
		Short:   "Creates a helm mirror ",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory which contains the charts/repositories.yml file")
	cmd.Flags().StringVarP(&o.Branch, "branch", "b", "gh-pages", "the git branch to clone the repository")
	cmd.Flags().StringVarP(&o.RepoURL, "url", "u", "", "the git URL of the repository to mirror the charts into")
	cmd.Flags().StringVarP(&o.CommitMessage, "message", "m", "chore: upgrade mirrored charts", "the commit message")
	cmd.Flags().StringArrayVarP(&o.Excludes, "exclude", "x", []string{"jenkins-x", "jx3"}, "the helm repositories to exclude from mirroring")
	cmd.Flags().StringVarP(&o.RepoType, "repo-type", "t", "Git", "the type of repository (Git or ChartMuseum)")

	o.Factory.AddFlags(cmd)
	return cmd, o
}

// Validate the arguments
func (o *Options) Validate() error {
	if o.RepoURL == "" {
		return options.MissingOption("url")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}

	if o.GitToken == "" {
		if o.GitServerURL == "" {
			gitInfo, err := giturl.ParseGitURL(o.RepoURL)
			if err != nil {
				return errors.Wrapf(err, "failed to parse git URL %s", o.RepoURL)
			}
			o.GitServerURL = gitInfo.HostURL()
		}

		err := o.Factory.FindGitToken()
		if err != nil {
			return errors.Wrapf(err, "failed to find git token")
		}
		if o.GitToken == "" {
			return options.MissingOption("git-token")
		}
	}
	// TODO: Validate the ChartMuseum Types with a JX Helper
	if o.RepoType != "Git" && o.RepoType != "ChartMuseum" {
		return errors.Errorf("unknown repository type %s, expected 'Git' or 'ChartMuseum'", o.RepoType)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}
	prefixes, err := versionstream.GetRepositoryPrefixes(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load quickstart repositories")
	}
	if prefixes == nil {
		return errors.Errorf("no chart repository prefixes")
	}
	if len(prefixes.Repositories) == 0 {
		return errors.Errorf("could not find charts/repositories.yml file in dir %s", o.Dir)
	}

	// No push, return early
	if o.NoPush {
		log.Logger().Infof("NoPush is set, skipping push operation")
		return nil
	}

	// Call the appropriate helper based on RepoType
	switch o.RepoType {
	case "Git":
		return o.pushChartToGit(prefixes)
	case "ChartMuseum":
		return o.pushChartToMuseum(prefixes)
	default:
		return errors.Errorf("unknown repository type %s, expected 'Git' or 'ChartMuseum'", o.RepoType)
	}
}

// pushChartToGit push chart to git repository
func (o *Options) pushChartToGit(prefixes *versionstream.RepositoryPrefixes) error {
	gitDir, err := ghpages.CloneGitHubPagesToDir(o.GitClient, o.RepoURL, o.Branch, o.GitUsername, o.GitToken)
	if err != nil {
		return errors.Wrapf(err, "failed to clone the GitHub pages repo %s branch %s", o.RepoURL, o.Branch)
	}
	if gitDir == "" {
		return errors.Errorf("no GitHub pages clone dir")
	}
	log.Logger().Infof("cloned GitHub pages repository to %s", info(gitDir))

	for _, repo := range prefixes.Repositories {
		name := repo.Prefix
		if stringhelpers.StringArrayIndex(o.Excludes, name) >= 0 {
			continue
		}
		outDir := filepath.Join(gitDir, name)
		err = os.MkdirAll(outDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create dir %s", outDir)
		}

		err = o.MirrorRepository(outDir, repo.URLs)
		if err != nil {
			return errors.Wrapf(err, "failed to mirror repository %s", name)
		}
	}

	changes, err := gitclient.AddAndCommitFiles(o.GitClient, gitDir, o.CommitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to add and commit files")
	}
	if !changes {
		log.Logger().Infof("no changes")
		return nil
	}

	err = gitclient.Pull(o.GitClient, gitDir)
	if err != nil {
		return errors.Wrapf(err, "failed to push changes")
	}

	log.Logger().Infof("pushed changes to %s in branch %s", info(o.GitURL), info(o.Branch))
	return nil
}

// MirrorRepository downloads the index yaml and all the referenced charts to the given directory
func (o *Options) MirrorRepository(dir string, urls []string) error {
	for _, u := range urls {
		path := filepath.Join(dir, "index.yaml")
		indexURL := stringhelpers.UrlJoin(u, "index.yaml")
		err := downloadURLToFile(indexURL, path)
		if err != nil {
			log.Logger().Warnf("failed to download index for %s", indexURL)
			continue
		}

		idx, err := repo.LoadIndexFile(path)
		if err != nil {
			log.Logger().Warnf("failed to load index file at %s", path)
			return nil
		}

		log.Logger().Infof("downloaded %s", info(path))
		err = o.DownloadIndex(idx, u, dir)
		if err != nil {
			return errors.Wrapf(err, "failed to download index for %s", dir)
		}
	}
	return nil
}

// pushChartToMuseum uses the chartmuseum.Client to upload charts to a ChartMuseum instance
func (o *Options) pushChartToMuseum(prefixes *versionstream.RepositoryPrefixes) error {
	// Create a ChartMuseum client
	client, err := o.createChartMuseumClient()
	if err != nil {
		return errors.Wrap(err, "failed to create ChartMuseum client")
	}

	for _, repo := range prefixes.Repositories {
		name := repo.Prefix
		if stringhelpers.StringArrayIndex(o.Excludes, name) >= 0 {
			continue
		}
		outDir := filepath.Join(o.Dir, name)

		// Check if directory exists and contains Helm charts
		files, err := os.ReadDir(outDir)
		if err != nil {
			return errors.Wrapf(err, "failed to read directory %s", outDir)
		}

		// Iterate through each file (expecting `.tgz` Helm chart files)
		for _, file := range files {
			if filepath.Ext(file.Name()) == ".tgz" {
				chartPath := filepath.Join(outDir, file.Name())

				// Use the ChartMuseum client to upload the chart
				err := o.uploadChartToMuseum(client, chartPath)
				if err != nil {
					log.Logger().Warnf("failed to upload chart %s to ChartMuseum", chartPath)
					continue
				}

				log.Logger().Infof("successfully uploaded chart %s to ChartMuseum", chartPath)
			}
		}
	}
	return nil
}

// createChartMuseumClient initializes and returns a chartmuseum.Client
func (o *Options) createChartMuseumClient() (*chartmuseum.Client, error) {
	opts := []chartmuseum.Option{
		chartmuseum.URL(o.RepoURL), // RepoURL now includes the ChartMuseum URL
	}

	// Optionally, add username/password or token for authentication if provided
	if o.GitUsername != "" && o.GitToken != "" {
		opts = append(opts, chartmuseum.Username(o.GitUsername), chartmuseum.Password(o.GitToken))
	}

	// Create the ChartMuseum client
	client, err := chartmuseum.NewClient(opts...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create ChartMuseum client")
	}

	return client, nil
}

// uploadChartToMuseum uploads a chart file to the ChartMuseum instance using chartmuseum.Client
func (o *Options) uploadChartToMuseum(client *chartmuseum.Client, chartPath string) error {
	// Upload the chart using the ChartMuseum client
	resp, err := client.UploadChartPackage(chartPath, false) // false means "do not force upload"
	if err != nil {
		return errors.Wrapf(err, "failed to upload chart %s", chartPath)
	}

	// Check if the response is successful (e.g., status 201 Created)
	if resp.StatusCode != http.StatusCreated {
		return errors.Errorf("failed to upload chart %s, received status %s", chartPath, resp.Status)
	}

	return nil
}

func (o *Options) DownloadIndex(idx *repo.IndexFile, u, dir string) error {
	for _, v := range idx.Entries {
		for _, cv := range v {
			for _, name := range cv.URLs {
				path := filepath.Join(dir, name)
				exists, err := files.FileExists(path)
				if err != nil {
					return errors.Wrapf(err, "failed to check for path %s", path)
				}
				if exists {
					continue
				}

				fileURL := stringhelpers.UrlJoin(u, name)
				err = downloadURLToFile(fileURL, path)
				if err != nil {
					log.Logger().Warnf("failed to download %s", fileURL)
					continue
				}
				log.Logger().Infof("downloaded %s", info(path))
			}
		}
	}
	return nil
}

func downloadURLToFile(u, path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}

	client := httphelpers.GetClient()
	req, err := http.NewRequest("GET", u, http.NoBody)
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", u)
	}

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return errors.Wrapf(err, "failed to GET endpoint %s with status %s", u, resp.Status)
		}
		return errors.Wrapf(err, "failed to GET endpoint %s", u)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read response from %s", u)
	}

	err = os.WriteFile(path, body, files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}
	return nil
}
