package integration

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sapk/docker-volume-gluster/gluster"
	"github.com/sapk/docker-volume-gluster/gluster/driver"
)

const timeInterval = 5 * time.Second

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
	driver.CfgFolder = "/etc/docker-volumes/" + gluster.PluginAlias
	log.Print(cmd("rm", "-rf", driver.CfgFolder))
	log.SetLevel(log.DebugLevel)
	gluster.PluginAlias = "gluster-local-integration"
	gluster.BaseDir = filepath.Join(volume.DefaultDockerRootDirectory, gluster.PluginAlias)
	gluster.DaemonStart(nil, []string{})
	time.Sleep(timeInterval)
	//log.Print(cmd("docker", "plugin", "ls"))
	log.Print(cmd("docker", "info", "-f", "{{.Plugins.Volume}}"))
}

func setupGlusterCluster() {
	pwd := currentPWD()
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "up", "-d"))
	time.Sleep(timeInterval)
	nodes := []string{"node-1", "node-2", "node-3"}

	for _, n := range nodes {
		time.Sleep(timeInterval)
		log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", n, "mkdir", "-p", "/brick"))
		/*
			time.Sleep(1 * time.Second)
			log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", n, "mkdir", "-p", "/brick/replica"))
			time.Sleep(1 * time.Second)
			log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", n, "mkdir", "-p", "/brick/distributed"))
		*/
	}

	containers := getGlusterClusterContainers()
	log.Print("CIDs : ", containers)
	for _, n := range []int{1, 2} {
		time.Sleep(timeInterval)
		ip := getContainerIP(containers[n])
		log.Print("IP node-"+strconv.Itoa(n+1)+" : ", ip)
		log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "peer", "probe", ip))
	}

	time.Sleep(timeInterval)
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "pool", "list"))
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "peer", "status"))
	time.Sleep(timeInterval)
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "create", "test-replica", "replica", "3", "node-1:/brick/replica", "node-2:/brick/replica", "node-3:/brick/replica"))
	time.Sleep(timeInterval)
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "create", "test-distributed", "node-1:/brick/distributed", "node-2:/brick/distributed", "node-3:/brick/distributed"))

	time.Sleep(timeInterval)
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "start", "test-replica"))
	time.Sleep(timeInterval)
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "exec", "-T", "node-1", "gluster", "volume", "start", "test-distributed"))
	time.Sleep(timeInterval)
}

//gluster pool list
func cleanGlusterCluster() {
	pwd := currentPWD()
	log.Print(cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "down"))
	log.Print(cmd("docker", "volume", "rm", "-f", "distributed", "replica"))
	log.Print(cmd("docker", "volume", "prune", "-f"))
	//TODO log.Print(cmd("docker", "system", "prune", "-af"))
}

func cmd(cmd string, arg ...string) (string, error) {
	fmt.Println("Executing: " + cmd + " " + strings.Join(arg, " "))
	c := exec.Command(cmd, arg...)
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	err := c.Run()
	return out.String(), err
}

func currentPWD() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func getGlusterClusterContainers() []string {
	pwd := currentPWD()
	list, _ := cmd("docker-compose", "-f", pwd+"/docker/gluster-cluster/docker-compose.yml", "ps", "-q")
	return strings.Split(list, "\n")
}

func getContainerIP(cid string) string {
	ips, _ := cmd("docker", "inspect", "--format", "'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'", cid)
	return strings.Trim(strings.Split(ips, "\n")[0], "'")
}

func TestIntegration(t *testing.T) {
	//Startplugin with empty config
	go setupPlugin()
	time.Sleep(3 * timeInterval)

	containers := getGlusterClusterContainers()
	log.Print("CIDs : ", containers)
	ip := getContainerIP(containers[0])
	log.Print("IP node-1 : ", ip)

	log.Print(cmd("docker", "volume", "create", "--driver", gluster.PluginAlias, "--opt", "voluri=\""+ip+":test-replica\"", "replica"))
	time.Sleep(timeInterval)
	log.Print(cmd("docker", "volume", "create", "--driver", gluster.PluginAlias, "--opt", "voluri=\""+ip+":test-distributed\"", "distributed"))
	time.Sleep(timeInterval)
	log.Print(cmd("docker", "volume", "ls"))
	time.Sleep(3 * timeInterval)
	//TODO docker volume create --driver sapk/plugin-gluster --opt voluri="<volumeserver>:<volumename>" --name test

	log.Print(cmd("docker", "run", "--rm", "-t", "-v", "replica:/mnt", "alpine", "/bin/ls", "/mnt"))
	log.Print(cmd("docker", "run", "--rm", "-t", "-v", "replica:/mnt", "alpine", "/bin/cp", "/etc/hostname", "/mnt/container"))
	log.Print(cmd("docker", "run", "--rm", "-t", "-v", "replica:/mnt", "alpine", "/bin/cat", "/mnt/container"))

	time.Sleep(3 * timeInterval)
	log.Print(cmd("docker", "run", "--rm", "-t", "-v", "distributed:/mnt", "alpine", "/bin/ls", "/mnt"))
	log.Print(cmd("docker", "run", "--rm", "-t", "-v", "distributed:/mnt", "alpine", "/bin/cp", "/etc/hostname", "/mnt/container"))
	log.Print(cmd("docker", "run", "--rm", "-t", "-v", "distributed:/mnt", "alpine", "/bin/cat", "/mnt/container"))
}
