package newdev

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"github.com/openshift/openshift-azure/hack/devtool/version"
	pluginapi "github.com/openshift/openshift-azure/pkg/api/plugin"
	"github.com/openshift/openshift-azure/pkg/util/pluginversion"
)

// NewCommand returns the cobra command for "dev-version".
func NewCommand() *cobra.Command {
	return &cobra.Command{
		Use:  "newdev",
		Long: "Start the new development version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return start(cmd)
		},
	}
}

func updatePluginconfig() (string, string, error) {
	b, err := ioutil.ReadFile("pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		return "", "", err
	}
	var template *pluginapi.Config
	err = yaml.Unmarshal(b, &template)
	if err != nil {
		return "", "", err
	}
	currentVersion := template.PluginVersion
	major, _, err := pluginversion.Parse(currentVersion)
	if err != nil {
		return "", "", err
	}
	nextVersion := fmt.Sprintf("v%d.0", major+1)

	// copy current section to next section
	template.Versions[nextVersion] = template.Versions[currentVersion]

	azureCI := "quay.io/openshift-on-azure/ci-azure:latest"
	next := template.Versions[nextVersion]
	next.Images.AzureControllers = azureCI
	next.Images.Canary = azureCI
	next.Images.EtcdBackup = azureCI
	next.Images.MetricsBridge = azureCI
	next.Images.Startup = azureCI
	next.Images.Sync = azureCI
	next.Images.TLSProxy = azureCI
	template.Versions[nextVersion] = next

	azureAcr := "osarpint.azurecr.io/openshift-on-azure/azure:" + currentVersion
	current := template.Versions[currentVersion]
	current.Images.AzureControllers = azureAcr
	current.Images.Canary = azureAcr
	current.Images.EtcdBackup = azureAcr
	current.Images.MetricsBridge = azureAcr
	current.Images.Startup = azureAcr
	current.Images.Sync = azureAcr
	current.Images.TLSProxy = azureAcr
	template.Versions[currentVersion] = current

	b, err = yaml.Marshal(template)
	if err != nil {
		return nextVersion, currentVersion, err
	}
	return nextVersion, currentVersion, ioutil.WriteFile("pluginconfig/pluginconfig-311.yaml", b, 0644)
}

func copy(src, dest string, info os.FileInfo) error {
	if info.IsDir() {
		return dcopy(src, dest, info)
	}
	return fcopy(src, dest, info)
}

func fcopy(src, dest string, info os.FileInfo) error {
	if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
		return err
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = os.Chmod(f.Name(), info.Mode()); err != nil {
		return err
	}
	s, err := os.Open(src) // #nosec G304
	if err != nil {
		return err
	}
	defer s.Close()
	_, err = io.Copy(f, s)
	return err
}

func dcopy(srcdir, destdir string, info os.FileInfo) error {
	if err := os.MkdirAll(destdir, info.Mode()); err != nil {
		return err
	}
	contents, err := ioutil.ReadDir(srcdir)
	if err != nil {
		return err
	}
	for _, content := range contents {
		cs, cd := filepath.Join(srcdir, content.Name()), filepath.Join(destdir, content.Name())
		if err := copy(cs, cd, content); err != nil {
			return err
		}
	}
	return nil
}

func copyVersionedDirs(newVer, oldVer string) error {
	oldVer, err := version.PluginToDevVersion(oldVer)
	if err != nil {
		return err
	}

	for _, pkg := range []string{"startup", "sync", "arm", "config"} {
		src := path.Join("pkg", pkg, oldVer)
		dest := path.Join("pkg", pkg, newVer[:len(newVer)-2]) // remove the ".0"
		info, err := os.Lstat(src)
		if err != nil {
			return err
		}
		err = copy(src, dest, info)
		if err != nil {
			return err
		}
		err = gitAdd(dest)
		if err != nil {
			return err
		}
	}
	return nil
}

func gitAdd(path string) error {
	return gitCommand([]string{"add", path})
}

func gitCommit(msg string) error {
	return gitCommand([]string{"commit", "-m", msg})
}

func gitCommand(args []string) error {
	cmd := exec.Command("git", args...) // #nosec G204
	fmt.Printf("git %v\n", args)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return cmd.Wait()
}

func start(cc *cobra.Command) error {
	newVer, oldVer, err := updatePluginconfig()
	if err != nil {
		return err
	}
	err = gitAdd("pluginconfig/pluginconfig-311.yaml")
	if err != nil {
		return err
	}
	err = gitCommit("Update pluginconfig for new dev version " + newVer)
	if err != nil {
		return err
	}
	err = copyVersionedDirs(newVer, oldVer)
	if err != nil {
		return err
	}
	err = gitCommit("Add new development versioned directories for " + newVer)
	if err != nil {
		return err
	}
	fmt.Println("TODO: fix version switch statements in the following files")
	for _, pkg := range []string{"startup", "sync", "arm", "config"} {
		fmt.Println(path.Join("pkg", pkg, newVer[:len(newVer)-2], pkg+".go"))
	}
	return nil
}
