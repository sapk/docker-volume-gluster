package integration

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/go-plugins-helpers/volume"
	"github.com/sapk/docker-volume-gluster/gluster"
	"github.com/sapk/docker-volume-gluster/gluster/driver"
	"github.com/sirupsen/logrus"
)

const timeInterval = 2 * time.Second

var randomTestName = "000000"

func TestMain(m *testing.M) {
	//TODO check system for gluster, docker and docker-compose (install container version if needed)
	//TODO need root to start plugin

	randomTestName = strconv.Itoa(rand.Int())

	//Setup cluster
	setupGlusterCluster()
	//Clean up
	defer cleanGlusterCluster()

	//Do tests
	//retVal := m.Run()
	m.Run()

	//os.Exit(retVal)
}

func setupPlugin() {
	gluster.PluginAlias = "gluster-local-integration"
	gluster.BaseDir = filepath.Join(volume.DefaultDockerRootDirectory, gluster.PluginAlias)
	driver.CfgFolder = "/etc/docker-volumes/" + gluster.PluginAlias
	logrus.Print(cmd("rm", os.Environ(), "-rf", driver.CfgFolder))
	logrus.SetLevel(logrus.DebugLevel)

	gluster.DaemonStart(nil, []string{})
	time.Sleep(timeInterval)
	//logrus.Print(cmd("docker", "plugin", "ls"))
	logrus.Print(cmd("docker", os.Environ(), "info", "-f", "{{.Plugins.Volume}}"))
}

func setupManagedPlugin() {
	//Build custom integration docker plugin
	env := os.Environ()
	env = append(env, fmt.Sprintf("PLUGIN_USER=%s", "testing"))
	logrus.Print(cmd("make", env, "--directory=../../", "docker-plugin"))
	logrus.Print(cmd("make", env, "--directory=../../", "docker-plugin-enable"))
}

func cleanManagedPlugin() {
	logrus.Print(cmd("docker", os.Environ(), "plugin", "disable", "testing/plugin-gluster"))
	logrus.Print(cmd("docker", os.Environ(), "plugin", "rm", "testing/plugin-gluster"))
}

func setupGlusterCluster() {
	logrus.Print(dockerCompose("up", "-d"))
	time.Sleep(timeInterval)
	nodes := []string{"node-1", "node-2", "node-3"}

	for _, n := range nodes {
		time.Sleep(timeInterval)
		logrus.Print(dockerCompose("exec", "-T", n, "mkdir", "-p", "/brick"))
	}

	time.Sleep(timeInterval)
	IPs := getGlusterClusterContainersIPs()
	for _, ip := range IPs[1:] {
		time.Sleep(timeInterval)
		logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "peer", "probe", ip))
	}

	time.Sleep(timeInterval)
	logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "pool", "list"))
	logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "peer", "status"))
	time.Sleep(timeInterval)

	logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "volume", "create", "test-replica", "replica", "3", IPs[0]+":/brick/replica", IPs[1]+":/brick/replica", IPs[2]+":/brick/replica"))
	time.Sleep(timeInterval)
	logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "volume", "create", "test-distributed", IPs[0]+":/brick/distributed", IPs[1]+":/brick/distributed", IPs[2]+":/brick/distributed"))

	time.Sleep(timeInterval)
	logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "volume", "start", "test-replica"))
	time.Sleep(timeInterval)
	logrus.Print(dockerCompose("exec", "-T", "node-1", "gluster", "volume", "start", "test-distributed"))
	time.Sleep(timeInterval)
}

func cleanGlusterCluster() {
	logrus.Print("Cleaning up")
	logrus.Print(dockerCompose("down"))
	//logrus.Print(cmd("docker", "volume", "rm", "-f", "glustercluster_brick-node-2", "glustercluster_brick-node-1", "glustercluster_brick-node-3", "glustercluster_state-node-1", "glustercluster_state-node-2", "glustercluster_state-node-3"))
	// extra clean up logrus.Print(cmd("docker", "volume", "prune", "-f"))
	// full  extra clean up logrus.Print(cmd("docker", "system", "prune", "-af"))
}

func dockerCompose(arg ...string) (string, error) {
	pwd := currentPWD()
	args := append([]string{"-f", pwd + "/docker/gluster-cluster/docker-compose.yml", "--project-name", "test_inte_" + randomTestName}, arg...)
	return cmd("docker-compose", os.Environ(), args...)
}

func cmd(cmd string, env []string, arg ...string) (string, error) {
	fmt.Println("Executing: " + cmd + " " + strings.Join(arg, " "))
	c := exec.Command(cmd, arg...)
	c.Env = env
	var out bytes.Buffer
	c.Stdout = &out
	c.Stderr = &out
	err := c.Run()
	return out.String(), err
}

//TODO cache result ?
func currentPWD() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Dir(filename)
}

func getGlusterClusterContainers() []string {
	list, _ := dockerCompose("ps", "-q")
	l := strings.Split(list, "\n")
	return l[:len(l)-1]
}

func getContainerIP(cid string) string {
	ips, _ := cmd("docker", os.Environ(), "inspect", "--format", "'{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'", cid)
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

	if !testing.Short() {
		//Setup managed plugin
		setupManagedPlugin()
		//Clean up
		defer cleanManagedPlugin()
	}

	IPs := getGlusterClusterContainersIPs()

	testCases := []struct {
		id       string
		name     string
		driver   string
		volume   string
		servers  []string
		hostname string
	}{
		{strconv.Itoa(rand.Int()), "replica", gluster.PluginAlias, "test-replica", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "distributed", gluster.PluginAlias, "test-distributed", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "replica-double-server", gluster.PluginAlias, "test-replica", IPs[:2], ""},
		{strconv.Itoa(rand.Int()), "distributed-double-server", gluster.PluginAlias, "test-distributed", IPs[:2], ""},
		{strconv.Itoa(rand.Int()), "replica-subdir", gluster.PluginAlias, "test-replica/subdir", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "distributed-subdir", gluster.PluginAlias, "test-distributed/subdir", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "managed-replica", "testing/plugin-gluster", "test-replica", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "managed-distributed", "testing/plugin-gluster", "test-distributed", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "managed-replica-double-server", "testing/plugin-gluster", "test-replica", IPs[:2], ""},
		{strconv.Itoa(rand.Int()), "managed-distributed-double-server", "testing/plugin-gluster", "test-distributed", IPs[:2], ""},
		{strconv.Itoa(rand.Int()), "managed-replica-subdir", gluster.PluginAlias, "test-replica/subdir", IPs[:1], ""},
		{strconv.Itoa(rand.Int()), "managed-distributed-subdir", gluster.PluginAlias, "test-distributed/subdir", IPs[:1], ""},
	}

	if testing.Short() { //Disable managed volume
		testCases = []struct {
			id       string
			name     string
			driver   string
			volume   string
			servers  []string
			hostname string
		}{
			{strconv.Itoa(rand.Int()), "replica", gluster.PluginAlias, "test-replica", IPs[:1], ""},
			{strconv.Itoa(rand.Int()), "distributed", gluster.PluginAlias, "test-distributed", IPs[:1], ""},
			{strconv.Itoa(rand.Int()), "replica-double-server", gluster.PluginAlias, "test-replica", IPs[:2], ""},
			{strconv.Itoa(rand.Int()), "distributed-double-server", gluster.PluginAlias, "test-distributed", IPs[:2], ""},
			{strconv.Itoa(rand.Int()), "replica-subdir", gluster.PluginAlias, "test-replica/subdir", IPs[:1], ""},
			{strconv.Itoa(rand.Int()), "distributed-subdir", gluster.PluginAlias, "test-distributed/subdir", IPs[:1], ""},
		}
	}

	for _, tc := range testCases {
		t.Run("Create volume for "+tc.name, func(t *testing.T) {
			logrus.Print(cmd("docker", os.Environ(), "volume", "create", "--driver", tc.driver, "--opt", "voluri=\""+strings.Join(tc.servers, ",")+":"+tc.volume+"\"", tc.id))
			time.Sleep(timeInterval)
		})
		time.Sleep(3 * timeInterval)
		//TODO test volume exist
	}

	logrus.Print(cmd("docker", os.Environ(), "volume", "ls"))

	for i, tc := range testCases {
		t.Run("Test volume "+tc.name, func(t *testing.T) {
			out, err := cmd("docker", os.Environ(), "run", "--rm", "-t", "-v", tc.id+":/mnt", "alpine", "/bin/ls", "/mnt")
			logrus.Println(out)
			if err != nil {
				t.Errorf("Failed to list mounted volume : %v", err)
			}
			if !strings.Contains(tc.name, "double") && !strings.Contains(tc.name, "managed") {
				out, err = cmd("docker", os.Environ(), "run", "--rm", "-t", "-v", tc.id+":/mnt", "alpine", "/bin/cp", "/etc/hostname", "/mnt/container")
				logrus.Println(out)
				if err != nil {
					t.Errorf("Failed to write inside mounted volume : %v", err)
				}
				out, err = cmd("docker", os.Environ(), "run", "--rm", "-t", "-v", tc.id+":/mnt", "alpine", "/bin/mkdir", "/mnt/subdir")
				logrus.Println(out)
				if err != nil {
					t.Errorf("Failed to create dir inside mounted volume : %v", err)
				}
				out, err = cmd("docker", os.Environ(), "run", "--rm", "-t", "-v", tc.id+":/mnt", "alpine", "/bin/cp", "/etc/hostname", "/mnt/subdir/container")
				logrus.Println(out)
				if err != nil {
					t.Errorf("Failed to write inside mounted volume : %v", err)
				}
			}
			testCases[i].hostname, err = cmd("docker", os.Environ(), "run", "--rm", "-t", "-v", tc.id+":/mnt", "alpine", "/bin/cat", "/mnt/container")
			logrus.Println(out)
			if err != nil {
				t.Errorf("Failed to read from mounted volume : %v", err)
			}
			time.Sleep(3 * timeInterval)
			//TODO check content is same
		})
	}

	for _, tc := range testCases {
		for _, td := range testCases {
			if tc.volume == td.volume {
				t.Run(fmt.Sprintf("Test same volume data between %s and %s", tc.name, td.name), func(t *testing.T) {
					if tc.hostname != td.hostname {
						t.Errorf("Content inside gluster %s volume in not the same : %s != %s", tc.volume, tc.hostname, td.hostname)
					}
				})
			}
		}
	}
	//TODO check persistence

	for _, tc := range testCases {
		out, err := cmd("docker", os.Environ(), "volume", "rm", tc.id)
		if err != nil {
			t.Errorf("Failed to remove mounted volume %s (%s) : %v", tc.name, tc.id, err)
		}
		if !strings.Contains(out, tc.id) { //TODO should be only "vol\n"
			t.Errorf("Failed to remove mounted volume %s (%s)", tc.name, tc.id)
		}
		out, err = cmd("docker", os.Environ(), "volume", "ls", "-q")
		if strings.Contains(out, tc.id) { //TODO should be "vol\n" to limit confussion ith other volume existing or generate name
			t.Errorf("Failed to remove volume %s (%s) from volume list", tc.name, tc.id)
		}
		time.Sleep(3 * timeInterval)
	}

}
