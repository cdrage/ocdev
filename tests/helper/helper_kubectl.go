package helper

import (
	"fmt"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

type KubectlRunner struct {
	// path to kubectl binary
	path string
}

// NewKubectlRunner initializes new KubectlRunner
func NewKubectlRunner(kubectlPath string) KubectlRunner {
	return KubectlRunner{
		path: kubectlPath,
	}
}

// Run kubectl with given arguments
func (kubectl KubectlRunner) Run(cmd string) *gexec.Session {
	session := CmdRunner(cmd)
	Eventually(session).Should(gexec.Exit(0))
	return session
}

// Exec allows generic execution of commands, returning the contents of stdout
func (kubectl KubectlRunner) Exec(podName string, projectName string, args ...string) string {

	cmd := []string{"exec", podName, "--namespace", projectName}

	cmd = append(cmd, args...)

	stdOut := CmdShouldPass(kubectl.path, cmd...)
	return stdOut
}

// ExecListDir returns dir list in specified location of pod
func (kubectl KubectlRunner) ExecListDir(podName string, projectName string, dir string) string {
	stdOut := CmdShouldPass(kubectl.path, "exec", podName, "--namespace", projectName,
		"--", "ls", "-lai", dir)
	return stdOut
}

// CheckCmdOpInRemoteDevfilePod runs the provided command on remote component pod and returns the return value of command output handler function passed to it
func (kubectl KubectlRunner) CheckCmdOpInRemoteDevfilePod(podName string, containerName string, prjName string, cmd []string, checkOp func(cmdOp string, err error) bool) bool {
	var execOptions []string
	execOptions = []string{"exec", podName, "--namespace", prjName, "--"}
	if containerName != "" {
		execOptions = []string{"exec", podName, "-c", containerName, "--namespace", prjName, "--"}
	}
	args := append(execOptions, cmd...)
	session := CmdRunner(kubectl.path, args...)
	stdOut := string(session.Wait().Out.Contents())
	stdErr := string(session.Wait().Err.Contents())
	if stdErr != "" && session.ExitCode() != 0 {
		return checkOp(stdOut, fmt.Errorf("cmd %s failed with error %s on pod %s", cmd, stdErr, podName))
	}
	return checkOp(stdOut, nil)
}

// GetRunningPodNameByComponent executes kubectl command and returns the running pod name of a delopyed
// devfile component by passing component name as a argument
func (kubectl KubectlRunner) GetRunningPodNameByComponent(compName string, namespace string) string {
	selector := fmt.Sprintf("--selector=component=%s", compName)
	stdOut := CmdShouldPass(kubectl.path, "get", "pods", "--namespace", namespace, selector, "-o", "jsonpath={.items[*].metadata.name}")
	return strings.TrimSpace(stdOut)
}

// GetPVCSize executes kubectl command and returns the bound storage size
func (kubectl KubectlRunner) GetPVCSize(compName, storageName, namespace string) string {
	selector := fmt.Sprintf("--selector=storage-name=%s,component=%s", storageName, compName)
	stdOut := CmdShouldPass(kubectl.path, "get", "pvc", "--namespace", namespace, selector, "-o", "jsonpath={.items[*].spec.resources.requests.storage}")
	return strings.TrimSpace(stdOut)
}

// GetPodInitContainers executes kubectl command and returns the init containers of the pod
func (kubectl KubectlRunner) GetPodInitContainers(compName string, namespace string) []string {
	selector := fmt.Sprintf("--selector=component=%s", compName)
	stdOut := CmdShouldPass(kubectl.path, "get", "pods", "--namespace", namespace, selector, "-o", "jsonpath={.items[*].spec.initContainers[*].name}")
	return strings.Split(stdOut, " ")
}

// GetVolumeMountNamesandPathsFromContainer returns the volume name and mount path in the format name:path\n
func (kubectl KubectlRunner) GetVolumeMountNamesandPathsFromContainer(deployName string, containerName, namespace string) string {
	volumeName := CmdShouldPass(kubectl.path, "get", "deploy", deployName, "--namespace", namespace,
		"-o", "go-template="+
			"{{range .spec.template.spec.containers}}{{if eq .name \""+containerName+
			"\"}}{{range .volumeMounts}}{{.name}}{{\":\"}}{{.mountPath}}{{\"\\n\"}}{{end}}{{end}}{{end}}")

	return strings.TrimSpace(volumeName)
}

// WaitAndCheckForExistence wait for the given and checks if the given resource type gets deleted on the cluster
func (kubectl KubectlRunner) WaitAndCheckForExistence(resourceType, namespace string, timeoutMinutes int) bool {
	pingTimeout := time.After(time.Duration(timeoutMinutes) * time.Minute)
	// this is a test package so time.Tick() is acceptable
	// nolint
	tick := time.Tick(time.Second)
	for {
		select {
		case <-pingTimeout:
			Fail(fmt.Sprintf("Timeout after %d minutes", timeoutMinutes))

		case <-tick:
			session := CmdRunner(kubectl.path, "get", resourceType, "--namespace", namespace)
			Eventually(session).Should(gexec.Exit(0))
			// https://github.com/kubernetes/kubectl/issues/847
			output := string(session.Wait().Err.Contents())

			if strings.Contains(strings.ToLower(output), "no resources found") {
				return true
			}
		}
	}
}

// GetServices gets services on the cluster
func (kubectl KubectlRunner) GetServices(namespace string) string {
	session := CmdRunner(kubectl.path, "get", "services", "--namespace", namespace)
	Eventually(session).Should(gexec.Exit(0))
	output := string(session.Wait().Out.Contents())
	return output
}

// CreateRandNamespaceProject create new project with random name in kubernetes cluster (10 letters)
func (kubectl KubectlRunner) CreateRandNamespaceProject() string {
	projectName := RandString(10)
	fmt.Fprintf(GinkgoWriter, "Creating a new project: %s\n", projectName)
	CmdShouldPass("kubectl", "create", "namespace", projectName)
	CmdShouldPass("kubectl", "config", "set-context", "--current", "--namespace", projectName)
	session := CmdShouldPass("kubectl", "get", "namespaces")
	Expect(session).To(ContainSubstring(projectName))
	return projectName
}

// DeleteNamespaceProject deletes a specified project in kubernetes cluster
func (kubectl KubectlRunner) DeleteNamespaceProject(projectName string) {
	fmt.Fprintf(GinkgoWriter, "Deleting project: %s\n", projectName)
	CmdShouldPass("kubectl", "delete", "namespaces", projectName)
}

func (kubectl KubectlRunner) GetEnvsDevFileDeployment(componentName string, projectName string) map[string]string {
	var mapOutput = make(map[string]string)

	output := CmdShouldPass(kubectl.path, "get", "deployment", componentName, "--namespace", projectName,
		"-o", "jsonpath='{range .spec.template.spec.containers[0].env[*]}{.name}:{.value}{\"\\n\"}{end}'")

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimPrefix(line, "'")
		splits := strings.Split(line, ":")
		name := splits[0]
		value := strings.Join(splits[1:], ":")
		mapOutput[name] = value
	}
	return mapOutput
}

func (kubectl KubectlRunner) GetAllPVCNames(namespace string) []string {
	session := CmdRunner(kubectl.path, "get", "pvc", "--namespace", namespace, "-o", "jsonpath={.items[*].metadata.name}")
	Eventually(session).Should(gexec.Exit(0))
	output := string(session.Wait().Out.Contents())
	if output == "" {
		return []string{}
	}
	return strings.Split(output, " ")
}
