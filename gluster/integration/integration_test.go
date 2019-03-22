package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sapk/docker-volume-gluster/gluster"
	"github.com/sapk/docker-volume-gluster/gluster/driver"
)

const timeInterval = 2 * time.Second

func TestMain(m *testing.M) {
	//TODO check system for gluster, docker and docker-compose (install container version if needed)
	//TODO need root to start plugin

	//Setup
	setupGlusterCluster()

	//Do tests
	retVal := m.Run()

	//Clean up
	cleanGlusterCluster()

	os.Exit(retVal)
}

func setupPlugin() {
	gluster.PluginAlias = "gluster-local-integration"
	gluster.BaseDir = filepath.Join(volume.DefaultDockerRootDirectory, gluster.PluginAlias)
	driver.CfgFolder = "/etc/docker-volumes/" + gluster.PluginAlias
	cmd("rm", "-rf", driver.CfgFolder)
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	gluster.DaemonStart(nil, []string{})
	time.Sleep(timeInterval)
	//cmd("docker", "plugin", "ls"))
	cmd("docker", "info", "-f", "{{.Plugins.Volume}}")
}

func setupGlusterCluster() {
	pwd := currentPWD()
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "up", "-d")
	time.Sleep(timeInterval)
	nodes := []string{"node-1", "node-2", "node-3"}

	for _, n := range nodes {
		time.Sleep(timeInterval)
		cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", n, "mkdir", "-p", "/brick")
	}

	time.Sleep(timeInterval)
	IPs := getGlusterClusterContainersIPs()
	for _, ip := range IPs[1:] {
		time.Sleep(timeInterval)
		cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "peer", "probe", ip)
	}

	time.Sleep(timeInterval)
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "pool", "list")
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "peer", "status")
	time.Sleep(timeInterval)

	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "create", "test-replica", "replica", "3", IPs[0]+":/brick/replica", IPs[1]+":/brick/replica", IPs[2]+":/brick/replica")
	time.Sleep(timeInterval)
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "create", "test-distributed", IPs[0]+":/brick/distributed", IPs[1]+":/brick/distributed", IPs[2]+":/brick/distributed")

	time.Sleep(timeInterval)
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "start", "test-replica")
	time.Sleep(timeInterval)
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "start", "test-distributed")
	time.Sleep(timeInterval)
}

//gluster pool list
func cleanGlusterCluster() {
	pwd := currentPWD()
	cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "down")
	cmd("docker", "volume", "rm", "-f", "distributed", "replica", "distributed-double-server", "replica-double-server")
	cmd("docker", "volume", "prune", "-f")
	//TODO cmd("docker", "system", "prune", "-af"))
}

func cmd(cmd string, arg ...string) (string, error) {
	fmt.Println("Executing: " + cmd + " " + strings.Join(arg, " "))
	c := exec.Command(cmd, arg...)
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	err := c.Run()
	outStr := out.String()
	fmt.Println(outStr)
	return outStr, err
}

func currentPWD() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func getGlusterClusterContainers() []string {
	pwd := currentPWD()
	list, _ := cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "ps", "-q")
	l := strings.Split(list, "\n")
	return l[:len(l)-1]
}

func getContainerIP(cid string) string {
	ips, _ := cmd("docker", "inspect", "--format", "'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'", cid)
	return strings.Trim(strings.Split(ips, "\n")[0], "'")
}

func getGlusterClusterContainersIPs() []string {
	containers := getGlusterClusterContainers()
	IPs := make([]string, len(containers))
	for i := range containers {
		IPs[i] = getContainerIP(containers[i])
	}
	return IPs
}

func TestIntegration(t *testing.T) {
	//Startplugin with empty config
	go setupPlugin()
	time.Sleep(3 * timeInterval)

	IPs := getGlusterClusterContainersIPs()

	cmd("docker", "volume", "create", "--driver", gluster.PluginAlias, "--opt", "voluri=\""+IPs[0]+":test-replica\"", "replica")
	time.Sleep(timeInterval)
	cmd("docker", "volume", "create", "--driver", gluster.PluginAlias, "--opt", "voluri=\""+IPs[0]+":test-distributed\"", "distributed")
	time.Sleep(timeInterval)
	cmd("docker", "volume", "create", "--driver", gluster.PluginAlias, "--opt", "voluri=\""+IPs[0]+","+IPs[1]+":test-replica\"", "replica-double-server")
	time.Sleep(timeInterval)
	cmd("docker", "volume", "create", "--driver", gluster.PluginAlias, "--opt", "voluri=\""+IPs[0]+","+IPs[1]+":test-distributed\"", "distributed-double-server")
	time.Sleep(timeInterval)
	cmd("docker", "volume", "ls")
	time.Sleep(3 * timeInterval)
	//TODO docker volume create --driver sapk/plugin-gluster --opt voluri="<volumeserver>:<volumename>" --name test

	out, err := cmd("docker", "run", "--rm", "-t", "-v", "replica:/mnt", "alpine", "/bin/ls", "/mnt")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to list mounted volume : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "replica:/mnt", "alpine", "/bin/cp", "/etc/hostname", "/mnt/container")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to write inside mounted volume : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "replica:/mnt", "alpine", "/bin/cat", "/mnt/container")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to read from mounted volume : %v", err)
	}
	time.Sleep(3 * timeInterval)

	out, err = cmd("docker", "run", "--rm", "-t", "-v", "distributed:/mnt", "alpine", "/bin/ls", "/mnt")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to list mounted volume : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "distributed:/mnt", "alpine", "/bin/cp", "/etc/hostname", "/mnt/container")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to write inside mounted volume : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "distributed:/mnt", "alpine", "/bin/cat", "/mnt/container")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to read from mounted volume : %v", err)
	}
	time.Sleep(3 * timeInterval)

	out, err = cmd("docker", "run", "--rm", "-t", "-v", "replica-double-server:/mnt", "alpine", "/bin/ls", "/mnt")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to list mounted volume (with fallback) : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "replica-double-server:/mnt", "alpine", "/bin/cat", "/mnt/container")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to read from mounted volume (with fallback) : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "distributed-double-server:/mnt", "alpine", "/bin/ls", "/mnt")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to list mounted volume (with fallback) : %v", err)
	}
	out, err = cmd("docker", "run", "--rm", "-t", "-v", "distributed-double-server:/mnt", "alpine", "/bin/cat", "/mnt/container")
	log.Info().Msg(out)
	if err != nil {
		t.Errorf("Failed to read from mounted volume (with fallback) : %v", err)
	}
	//TODO check that container is same as before

}
