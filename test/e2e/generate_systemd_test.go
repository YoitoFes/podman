package integration

import (
	"io/ioutil"
	"os"
	"strings"

	. "github.com/containers/podman/v4/test/utils"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gexec"
)

var _ = Describe("Podman generate systemd", func() {
	var (
		tempdir    string
		err        error
		podmanTest *PodmanTestIntegration
	)

	BeforeEach(func() {
		tempdir, err = CreateTempDirInTempDir()
		if err != nil {
			os.Exit(1)
		}
		podmanTest = PodmanTestCreate(tempdir)
		podmanTest.Setup()
	})

	AfterEach(func() {
		podmanTest.Cleanup()
		f := CurrentGinkgoTestDescription()
		processTestResult(f)

	})

	It("podman generate systemd on bogus container/pod", func() {
		session := podmanTest.Podman([]string{"generate", "systemd", "foobar"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())
	})

	It("podman generate systemd bad restart policy", func() {
		session := podmanTest.Podman([]string{"generate", "systemd", "--restart-policy", "never", "foobar"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())
	})

	It("podman generate systemd bad timeout value", func() {
		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "-1", "foobar"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())
	})

	It("podman generate systemd bad restart-policy value", func() {
		session := podmanTest.Podman([]string{"create", "--name", "foobar", "alpine", "top"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "--restart-policy", "bogus", "foobar"})
		session.WaitWithDefaultTimeout()
		Expect(session).To(ExitWithError())
		Expect(session.ErrorToString()).To(ContainSubstring("bogus is not a valid restart policy"))
	})

	It("podman generate systemd with --no-header=true", func() {
		session := podmanTest.Podman([]string{"create", "--name", "foobar", "alpine", "top"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "foobar", "--no-header=true"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		Expect(session.OutputToString()).NotTo(ContainSubstring("autogenerated by"))
	})

	It("podman generate systemd with --no-header", func() {
		session := podmanTest.Podman([]string{"create", "--name", "foobar", "alpine", "top"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "foobar", "--no-header"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		Expect(session.OutputToString()).NotTo(ContainSubstring("autogenerated by"))
	})

	It("podman generate systemd with --no-header=false", func() {
		session := podmanTest.Podman([]string{"create", "--name", "foobar", "alpine", "top"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "foobar", "--no-header=false"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		Expect(session.OutputToString()).To(ContainSubstring("autogenerated by"))
	})

	It("podman generate systemd good timeout value", func() {
		session := podmanTest.Podman([]string{"create", "--name", "foobar", "alpine", "top"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "--time", "1234", "foobar"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("TimeoutStopSec=1294"))
		Expect(session.OutputToString()).To(ContainSubstring(" stop -t 1234 "))
	})

	It("podman generate systemd", func() {
		n := podmanTest.Podman([]string{"run", "--name", "nginx", "-dt", NGINX_IMAGE})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The podman commands in the unit should not contain the root flags
		Expect(session.OutputToString()).ToNot(ContainSubstring(" --runroot"))
	})

	It("podman generate systemd --files --name", func() {
		n := podmanTest.Podman([]string{"run", "--name", "nginx", "-dt", NGINX_IMAGE})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--files", "--name", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		for _, file := range session.OutputToStringArray() {
			os.Remove(file)
		}
		Expect(session.OutputToString()).To(ContainSubstring("/container-nginx.service"))
	})

	It("podman generate systemd with timeout", func() {
		n := podmanTest.Podman([]string{"run", "--name", "nginx", "-dt", NGINX_IMAGE})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "5", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("TimeoutStopSec=65"))
		Expect(session.OutputToString()).ToNot(ContainSubstring("TimeoutStartSec="))
		Expect(session.OutputToString()).To(ContainSubstring("podman stop -t 5"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--stop-timeout", "5", "--start-timeout", "123", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("TimeoutStartSec=123"))
		Expect(session.OutputToString()).To(ContainSubstring("TimeoutStopSec=65"))
		Expect(session.OutputToString()).To(ContainSubstring("podman stop -t 5"))
	})

	It("podman generate systemd with user-defined dependencies", func() {
		n := podmanTest.Podman([]string{"run", "--name", "nginx", "-dt", NGINX_IMAGE})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--wants", "foobar.service", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined Wants option
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=foobar.service"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--after", "foobar.service", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined After option
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("After=foobar.service"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--requires", "foobar.service", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined Requires option
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("Requires=foobar.service"))

		session = podmanTest.Podman([]string{
			"generate", "systemd",
			"--wants", "foobar.service", "--wants", "barfoo.service",
			"--after", "foobar.service", "--after", "barfoo.service",
			"--requires", "foobar.service", "--requires", "barfoo.service", "nginx"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined Want, After, Requires options
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=foobar.service barfoo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("After=foobar.service barfoo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("Requires=foobar.service barfoo.service"))
	})

	It("podman generate systemd pod --name", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-2", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "42", "--name", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		output := session.OutputToString()
		Expect(output).To(ContainSubstring("# pod-foo.service"))
		Expect(output).To(ContainSubstring("Wants=container-foo-1.service container-foo-2.service"))
		Expect(output).To(ContainSubstring("# container-foo-1.service"))
		Expect(output).To(ContainSubstring(" start foo-1"))
		Expect(output).To(ContainSubstring("-infra")) // infra container
		Expect(output).To(ContainSubstring("# container-foo-2.service"))
		Expect(output).To(ContainSubstring(" stop -t 42 foo-2"))
		Expect(output).To(ContainSubstring("BindsTo=pod-foo.service"))
		Expect(output).To(ContainSubstring("PIDFile="))
		Expect(output).To(ContainSubstring("/userdata/conmon.pid"))
		Expect(strings.Count(output, "RequiresMountsFor="+podmanTest.RunRoot)).To(Equal(3))
		// The podman commands in the unit should not contain the root flags
		Expect(output).ToNot(ContainSubstring(" --runroot"))
	})

	It("podman generate systemd pod --name --files", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--name", "--files", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		for _, file := range session.OutputToStringArray() {
			os.Remove(file)
		}

		Expect(session.OutputToString()).To(ContainSubstring("/pod-foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("/container-foo-1.service"))
	})

	It("podman generate systemd pod with user-defined dependencies", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--name", "--wants", "foobar.service", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined Wants option
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=foobar.service"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--name", "--after", "foobar.service", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined After option
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("After=foobar.service"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--name", "--requires", "foobar.service", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined Requires option
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("Requires=foobar.service"))

		session = podmanTest.Podman([]string{
			"generate", "systemd", "--name",
			"--wants", "foobar.service", "--wants", "barfoo.service",
			"--after", "foobar.service", "--after", "barfoo.service",
			"--requires", "foobar.service", "--requires", "barfoo.service", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// The generated systemd unit should contain the User-defined Want, After, Requires options
		Expect(session.OutputToString()).To(ContainSubstring("# User-defined dependencies"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=foobar.service barfoo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("After=foobar.service barfoo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("Requires=foobar.service barfoo.service"))
	})

	It("podman generate systemd --new --name foo", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "-t", "42", "--name", "--new", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# container-foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring(" --replace "))
		if !IsRemote() {
			// The podman commands in the unit should contain the root flags if generate systemd --new is used
			Expect(session.OutputToString()).To(ContainSubstring(" --runroot"))
		}
	})

	It("podman generate systemd --new --name=foo", func() {
		n := podmanTest.Podman([]string{"create", "--name=foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "-t", "42", "--name", "--new", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# container-foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring(" --replace "))
	})

	It("podman generate systemd --new without explicit detaching param", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "42", "--name", "--new", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring(" -d "))
	})

	It("podman generate systemd --new with explicit detaching param in middle", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "42", "--name", "--new", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("--name foo alpine top"))
	})

	It("podman generate systemd --new pod", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "42", "--name", "--new", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring(" pod create "))
	})

	It("podman generate systemd --restart-sec 15 --name foo", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--restart-sec", "15", "--name", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("RestartSec=15"))
	})

	It("podman generate systemd --new=false pod", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "42", "--name", "--new=false", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).NotTo(ContainSubstring(" pod create "))
	})

	It("podman generate systemd --new=true pod", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--time", "42", "--name", "--new=true", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring(" pod create "))
	})

	It("podman generate systemd --container-prefix con", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--name", "--container-prefix", "con", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# con-foo.service"))

	})

	It("podman generate systemd --container-prefix ''", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--name", "--container-prefix", "", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# foo.service"))

	})

	It("podman generate systemd --separator _", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--name", "--separator", "_", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# container_foo.service"))
	})

	It("podman generate systemd pod --pod-prefix p", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-2", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--pod-prefix", "p", "--name", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# p-foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=container-foo-1.service container-foo-2.service"))
		Expect(session.OutputToString()).To(ContainSubstring("# container-foo-1.service"))
		Expect(session.OutputToString()).To(ContainSubstring("BindsTo=p-foo.service"))
	})

	It("podman generate systemd pod --pod-prefix p --container-prefix con --separator _ change all prefixes/separator", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-2", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--container-prefix", "con", "--pod-prefix", "p", "--separator", "_", "--name", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# p_foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=con_foo-1.service con_foo-2.service"))
		Expect(session.OutputToString()).To(ContainSubstring("# con_foo-1.service"))
		Expect(session.OutputToString()).To(ContainSubstring("# con_foo-2.service"))
		Expect(session.OutputToString()).To(ContainSubstring("BindsTo=p_foo.service"))
	})

	It("podman generate systemd pod --pod-prefix '' --container-prefix '' --separator _ change all prefixes/separator", func() {
		n := podmanTest.Podman([]string{"pod", "create", "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-2", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		// test systemd generate with empty pod prefix
		session1 := podmanTest.Podman([]string{"generate", "systemd", "--pod-prefix", "", "--name", "foo"})
		session1.WaitWithDefaultTimeout()
		Expect(session1).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session1.OutputToString()).To(ContainSubstring("# foo.service"))
		Expect(session1.OutputToString()).To(ContainSubstring("Wants=container-foo-1.service container-foo-2.service"))
		Expect(session1.OutputToString()).To(ContainSubstring("# container-foo-1.service"))
		Expect(session1.OutputToString()).To(ContainSubstring("BindsTo=foo.service"))

		// test systemd generate with empty container and pod prefix
		session2 := podmanTest.Podman([]string{"generate", "systemd", "--container-prefix", "", "--pod-prefix", "", "--separator", "_", "--name", "foo"})
		session2.WaitWithDefaultTimeout()
		Expect(session2).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session2.OutputToString()).To(ContainSubstring("# foo.service"))
		Expect(session2.OutputToString()).To(ContainSubstring("Wants=foo-1.service foo-2.service"))
		Expect(session2.OutputToString()).To(ContainSubstring("# foo-1.service"))
		Expect(session2.OutputToString()).To(ContainSubstring("# foo-2.service"))
		Expect(session2.OutputToString()).To(ContainSubstring("BindsTo=foo.service"))

	})

	It("podman generate systemd pod with containers --new", func() {
		tmpDir, err := ioutil.TempDir("", "")
		Expect(err).To(BeNil())
		tmpFile := tmpDir + "podID"
		defer os.RemoveAll(tmpDir)

		n := podmanTest.Podman([]string{"pod", "create", "--pod-id-file", tmpFile, "--name", "foo"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-1", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		n = podmanTest.Podman([]string{"create", "--pod", "foo", "--name", "foo-2", "alpine", "top"})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--new", "--name", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		// Grepping the output (in addition to unit tests)
		Expect(session.OutputToString()).To(ContainSubstring("# pod-foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("Wants=container-foo-1.service container-foo-2.service"))
		Expect(session.OutputToString()).To(ContainSubstring("BindsTo=pod-foo.service"))
		Expect(session.OutputToString()).To(ContainSubstring("pod create --infra-conmon-pidfile %t/pod-foo.pid --pod-id-file %t/pod-foo.pod-id --name foo"))
		Expect(session.OutputToString()).To(ContainSubstring("ExecStartPre=/bin/rm -f %t/pod-foo.pid %t/pod-foo.pod-id"))
		Expect(session.OutputToString()).To(ContainSubstring("pod stop --ignore --pod-id-file %t/pod-foo.pod-id -t 10"))
		Expect(session.OutputToString()).To(ContainSubstring("pod rm --ignore -f --pod-id-file %t/pod-foo.pod-id"))
	})

	It("podman generate systemd --format json", func() {
		n := podmanTest.Podman([]string{"create", "--name", "foo", ALPINE})
		n.WaitWithDefaultTimeout()
		Expect(n).Should(Exit(0))

		session := podmanTest.Podman([]string{"generate", "systemd", "--format", "json", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(BeValidJSON())
	})

	It("podman generate systemd --new create command with double curly braces", func() {
		SkipIfInContainer("journald inside a container doesn't work")
		// Regression test for #9034
		session := podmanTest.Podman([]string{"create", "--name", "foo", "--log-driver=journald", "--log-opt=tag={{.Name}}", ALPINE})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "--new", "foo"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring(" --log-opt=tag={{.Name}} "))

		session = podmanTest.Podman([]string{"pod", "create", "--name", "pod", "--label", "key={{someval}}"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "--new", "pod"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring(" --label key={{someval}}"))
	})

	It("podman generate systemd --env", func() {
		session := podmanTest.RunTopContainer("test")
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "foo=bar", "-e", "hoge=fuga", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("Environment=foo=bar"))
		Expect(session.OutputToString()).To(ContainSubstring("Environment=hoge=fuga"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "=bar", "-e", "hoge=fuga", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(125))
		Expect(session.ErrorToString()).To(ContainSubstring("invalid environment variable"))

		// Use -e/--env option with --new option
		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "foo=bar", "-e", "hoge=fuga", "--new", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("Environment=foo=bar"))
		Expect(session.OutputToString()).To(ContainSubstring("Environment=hoge=fuga"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "foo=bar", "-e", "=fuga", "--new", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(125))
		Expect(session.ErrorToString()).To(ContainSubstring("invalid environment variable"))

		// Escape systemd arguments
		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "BAR=my test", "-e", "USER=%a", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("\"BAR=my test\""))
		Expect(session.OutputToString()).To(ContainSubstring("USER=%%a"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "BAR=my test", "-e", "USER=%a", "--new", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("\"BAR=my test\""))
		Expect(session.OutputToString()).To(ContainSubstring("USER=%%a"))

		// Specify the environment variables without a value
		os.Setenv("FOO1", "BAR1")
		os.Setenv("FOO2", "BAR2")
		os.Setenv("FOO3", "BAR3")
		defer os.Unsetenv("FOO1")
		defer os.Unsetenv("FOO2")
		defer os.Unsetenv("FOO3")

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "FOO1", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("BAR1"))
		Expect(session.OutputToString()).NotTo(ContainSubstring("BAR2"))
		Expect(session.OutputToString()).NotTo(ContainSubstring("BAR3"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "FOO*", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("BAR1"))
		Expect(session.OutputToString()).To(ContainSubstring("BAR2"))
		Expect(session.OutputToString()).To(ContainSubstring("BAR3"))

		session = podmanTest.Podman([]string{"generate", "systemd", "--env", "FOO*", "--new", "test"})
		session.WaitWithDefaultTimeout()
		Expect(session).Should(Exit(0))
		Expect(session.OutputToString()).To(ContainSubstring("BAR1"))
		Expect(session.OutputToString()).To(ContainSubstring("BAR2"))
		Expect(session.OutputToString()).To(ContainSubstring("BAR3"))
	})
})
