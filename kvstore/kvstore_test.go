package kvstore

import (
	"os"
	"testing"
	"time"

	"github.com/rancher/longhorn-manager/types"
	"github.com/rancher/longhorn-manager/util"

	. "gopkg.in/check.v1"
)

const (
	TestPrefix = "longhorn-manager-test"

	EnvEtcdServer  = "LONGHORN_MANAGER_TEST_ETCD_SERVER"
	EnvEngineImage = "LONGHORN_ENGINE_IMAGE"
)

var (
	VolumeName     = TestPrefix + "-vol"
	ControllerName = VolumeName + "-controller"
	Replica1Name   = VolumeName + "-replica1"
	Replica2Name   = VolumeName + "-replica2"
	Replica3Name   = VolumeName + "-replica3"
	Replica4Name   = VolumeName + "-replica4"
)

func Test(t *testing.T) { TestingT(t) }

type TestSuite struct {
	s           *KVStore
	engineImage string
}

var _ = Suite(&TestSuite{})

func (s *TestSuite) SetUpTest(c *C) {
	var err error

	etcdIP := os.Getenv(EnvEtcdServer)
	c.Assert(etcdIP, Not(Equals), "")

	s.engineImage = os.Getenv(EnvEngineImage)
	c.Assert(s.engineImage, Not(Equals), "")

	store, err := NewKVStore([]string{"http://" + etcdIP + ":2379"}, "/longhorn")
	c.Assert(err, IsNil)
	s.s = store

	err = s.s.kvNuclear("nuke key value store")
	c.Assert(err, IsNil)
}

func (s *TestSuite) TeardownTest(c *C) {
	err := s.s.kvNuclear("nuke key value store")
	c.Assert(err, IsNil)
}

func (s *TestSuite) TestHost(c *C) {
	host1 := &types.HostInfo{
		UUID:    util.UUID(),
		Name:    "host-1",
		Address: "127.0.1.1",
	}
	host2 := &types.HostInfo{
		UUID:    util.UUID(),
		Name:    "host-2",
		Address: "127.0.1.2",
	}
	host3 := &types.HostInfo{
		UUID:    util.UUID(),
		Name:    "host-3",
		Address: "127.0.1.3",
	}

	err := s.s.SetHost(host1)
	c.Assert(err, IsNil)

	host, err := s.s.GetHost(host1.UUID)
	c.Assert(err, IsNil)
	c.Assert(host, DeepEquals, host1)

	host1.Address = "127.0.2.2"
	err = s.s.SetHost(host1)
	c.Assert(err, IsNil)

	host, err = s.s.GetHost(host1.UUID)
	c.Assert(err, IsNil)
	c.Assert(host, DeepEquals, host1)

	err = s.s.SetHost(host2)
	c.Assert(err, IsNil)

	err = s.s.SetHost(host3)
	c.Assert(err, IsNil)

	host, err = s.s.GetHost(host1.UUID)
	c.Assert(err, IsNil)
	c.Assert(host, DeepEquals, host1)

	host, err = s.s.GetHost(host2.UUID)
	c.Assert(err, IsNil)
	c.Assert(host, DeepEquals, host2)

	host, err = s.s.GetHost(host3.UUID)
	c.Assert(err, IsNil)
	c.Assert(host, DeepEquals, host3)

	hosts, err := s.s.ListHosts()
	c.Assert(err, IsNil)

	c.Assert(hosts[host1.UUID], DeepEquals, host1)
	c.Assert(hosts[host2.UUID], DeepEquals, host2)
	c.Assert(hosts[host3.UUID], DeepEquals, host3)

	host, err = s.s.GetHost("random")
	c.Assert(err, IsNil)
	c.Assert(host, IsNil)
}

func (s *TestSuite) TestSettings(c *C) {
	existing, err := s.s.GetSettings()
	c.Assert(err, IsNil)
	c.Assert(existing, IsNil)

	settings := &types.SettingsInfo{
		BackupTarget: "nfs://1.2.3.4:/test",
		EngineImage:  "rancher/longhorn",
	}

	err = s.s.SetSettings(settings)
	c.Assert(err, IsNil)

	newSettings, err := s.s.GetSettings()
	c.Assert(err, IsNil)
	c.Assert(newSettings.BackupTarget, Equals, settings.BackupTarget)
	c.Assert(newSettings.EngineImage, Equals, settings.EngineImage)
}

func generateTestVolume(name string) *types.VolumeInfo {
	return &types.VolumeInfo{
		Name:                name,
		Size:                1024 * 1024,
		NumberOfReplicas:    2,
		StaleReplicaTimeout: 1 * time.Minute,
	}
}

func generateTestController(volName string) *types.ControllerInfo {
	return &types.ControllerInfo{
		types.InstanceInfo{
			ID:         "controller-id-" + volName,
			Type:       types.InstanceTypeController,
			Name:       "controller-name-" + volName,
			Running:    true,
			VolumeName: volName,
		},
	}
}

func generateTestReplica(volName, replicaName string) *types.ReplicaInfo {
	return &types.ReplicaInfo{
		InstanceInfo: types.InstanceInfo{
			ID:         "replica-id-" + replicaName + "-" + volName,
			Type:       types.InstanceTypeReplica,
			Name:       "replica-name-" + replicaName + "-" + volName,
			Running:    true,
			VolumeName: volName,
		},
		Mode: types.ReplicaModeRW,
	}
}

func (s *TestSuite) verifyVolume(c *C, volume *types.VolumeInfo) {
	comp, err := s.s.GetVolume(volume.Name)
	c.Assert(err, IsNil)
	c.Assert(comp, DeepEquals, volume)
}

func (s *TestSuite) TestVolume(c *C) {
	var err error

	volume, err := s.s.GetVolume("random")
	c.Assert(err, IsNil)
	c.Assert(volume, IsNil)

	volume1 := generateTestVolume("volume1")
	controller1 := generateTestController(volume1.Name)
	replica11 := generateTestReplica(volume1.Name, "replica1")
	replica12 := generateTestReplica(volume1.Name, "replica2")
	volume1.Controller = controller1
	volume1.Replicas = map[string]*types.ReplicaInfo{
		replica11.Name: replica11,
		replica12.Name: replica12,
	}

	err = s.s.SetVolume(volume1)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume1)

	volume2 := generateTestVolume("volume2")
	controller2 := generateTestController(volume2.Name)
	replica21 := generateTestReplica(volume2.Name, "replica1")
	replica22 := generateTestReplica(volume2.Name, "replica2")
	volume2.Controller = controller2
	volume2.Replicas = map[string]*types.ReplicaInfo{
		replica21.Name: replica21,
		replica22.Name: replica22,
	}

	err = s.s.SetVolume(volume2)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	volumes, err := s.s.ListVolumes()
	c.Assert(err, IsNil)
	c.Assert(len(volumes), Equals, 2)

	volume2.Controller = nil
	err = s.s.SetVolume(volume2)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	volume2.Replicas = nil
	err = s.s.SetVolume(volume2)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	volume2.Replicas = map[string]*types.ReplicaInfo{
		replica21.Name: replica21,
	}
	err = s.s.SetVolume(volume2)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	volume2.Replicas[replica22.Name] = replica22
	err = s.s.SetVolume(volume2)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	volume2.Controller = controller2
	err = s.s.SetVolume(volume2)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	err = s.s.DeleteVolumeReplicas(volume2.Name)
	c.Assert(err, IsNil)
	volume2.Replicas = nil
	s.verifyVolume(c, volume2)

	err = s.s.SetVolumeReplica(replica21)
	c.Assert(err, IsNil)
	volume2.Replicas = map[string]*types.ReplicaInfo{
		replica21.Name: replica21,
	}
	s.verifyVolume(c, volume2)

	err = s.s.SetVolumeReplica(replica22)
	c.Assert(err, IsNil)
	volume2.Replicas[replica22.Name] = replica22
	s.verifyVolume(c, volume2)

	err = s.s.DeleteVolumeReplicas(volume2.Name)
	c.Assert(err, IsNil)
	volume2.Replicas = nil
	s.verifyVolume(c, volume2)

	volume2.Replicas = map[string]*types.ReplicaInfo{
		replica21.Name: replica21,
		replica22.Name: replica22,
	}
	err = s.s.SetVolumeReplicas(volume2.Replicas)
	c.Assert(err, IsNil)
	s.verifyVolume(c, volume2)

	err = s.s.DeleteVolumeController(volume2.Name)
	c.Assert(err, IsNil)
	volume2.Controller = nil
	s.verifyVolume(c, volume2)

	err = s.s.SetVolumeController(controller2)
	c.Assert(err, IsNil)
	volume2.Controller = controller2
	s.verifyVolume(c, volume2)

	err = s.s.DeleteVolume(volume1.Name)
	c.Assert(err, IsNil)

	volumes, err = s.s.ListVolumes()
	c.Assert(err, IsNil)
	c.Assert(len(volumes), Equals, 1)
	c.Assert(volumes[0], DeepEquals, volume2)

	err = s.s.DeleteVolume(volume2.Name)
	c.Assert(err, IsNil)

	volumes, err = s.s.ListVolumes()
	c.Assert(err, IsNil)
	c.Assert(len(volumes), Equals, 0)
}
