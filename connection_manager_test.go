package nebula

import (
	"crypto/ed25519"
	"crypto/rand"
	"net/netip"
	"testing"
	"time"

	"github.com/flynn/noise"
	"github.com/slackhq/nebula/cert"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/test"
	"github.com/slackhq/nebula/udp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestLighthouse() *LightHouse {
	lh := &LightHouse{
		l:         test.NewLogger(),
		addrMap:   map[netip.Addr]*RemoteList{},
		queryChan: make(chan netip.Addr, 10),
	}
	lighthouses := map[netip.Addr]struct{}{}
	staticList := map[netip.Addr]struct{}{}

	lh.lighthouses.Store(&lighthouses)
	lh.staticList.Store(&staticList)

	return lh
}

func Test_NewConnectionManagerTest(t *testing.T) {
	l := test.NewLogger()
	//_, tuncidr, _ := net.ParseCIDR("1.1.1.1/24")
	localrange := netip.MustParsePrefix("10.1.1.1/24")
	vpnIp := netip.MustParseAddr("172.1.1.2")
	preferredRanges := []netip.Prefix{localrange}

	// Very incomplete mock objects
	hostMap := newHostMap(l)
	hostMap.preferredRanges.Store(&preferredRanges)

	cs := &CertState{
		initiatingVersion: cert.Version1,
		privateKey:        []byte{},
		v1Cert:            &dummyCert{version: cert.Version1},
		v1HandshakeBytes:  []byte{},
	}

	lh := newTestLighthouse()
	ifce := &Interface{
		hostMap:          hostMap,
		inside:           &test.NoopTun{},
		outside:          &udp.NoopConn{},
		firewall:         &Firewall{},
		lightHouse:       lh,
		pki:              &PKI{},
		handshakeManager: NewHandshakeManager(l, hostMap, lh, &udp.NoopConn{}, defaultHandshakeConfig),
		l:                l,
	}
	ifce.pki.cs.Store(cs)

	// Create manager
	conf := config.NewC(l)
	punchy := NewPunchyFromConfig(l, conf)
	nc := newConnectionManagerFromConfig(l, conf, hostMap, punchy)
	nc.intf = ifce
	p := []byte("")
	nb := make([]byte, 12, 12)
	out := make([]byte, mtu)

	// Add an ip we have established a connection w/ to hostmap
	hostinfo := &HostInfo{
		vpnAddrs:      []netip.Addr{vpnIp},
		localIndexId:  1099,
		remoteIndexId: 9901,
	}
	hostinfo.ConnectionState = &ConnectionState{
		myCert: &dummyCert{version: cert.Version1},
		H:      &noise.HandshakeState{},
	}
	nc.hostMap.unlockedAddHostInfo(hostinfo, ifce)

	// We saw traffic out to vpnIp
	nc.Out(hostinfo)
	nc.In(hostinfo)
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
	assert.True(t, hostinfo.out.Load())
	assert.True(t, hostinfo.in.Load())

	// Do a traffic check tick, should not be pending deletion but should not have any in/out packets recorded
	nc.doTrafficCheck(hostinfo.localIndexId, p, nb, out, time.Now())
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())

	// Do another traffic check tick, this host should be pending deletion now
	nc.Out(hostinfo)
	assert.True(t, hostinfo.out.Load())
	nc.doTrafficCheck(hostinfo.localIndexId, p, nb, out, time.Now())
	assert.True(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])

	// Do a final traffic check tick, the host should now be removed
	nc.doTrafficCheck(hostinfo.localIndexId, p, nb, out, time.Now())
	assert.NotContains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs)
	assert.NotContains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
}

func Test_NewConnectionManagerTest2(t *testing.T) {
	l := test.NewLogger()
	//_, tuncidr, _ := net.ParseCIDR("1.1.1.1/24")
	localrange := netip.MustParsePrefix("10.1.1.1/24")
	vpnIp := netip.MustParseAddr("172.1.1.2")
	preferredRanges := []netip.Prefix{localrange}

	// Very incomplete mock objects
	hostMap := newHostMap(l)
	hostMap.preferredRanges.Store(&preferredRanges)

	cs := &CertState{
		initiatingVersion: cert.Version1,
		privateKey:        []byte{},
		v1Cert:            &dummyCert{version: cert.Version1},
		v1HandshakeBytes:  []byte{},
	}

	lh := newTestLighthouse()
	ifce := &Interface{
		hostMap:          hostMap,
		inside:           &test.NoopTun{},
		outside:          &udp.NoopConn{},
		firewall:         &Firewall{},
		lightHouse:       lh,
		pki:              &PKI{},
		handshakeManager: NewHandshakeManager(l, hostMap, lh, &udp.NoopConn{}, defaultHandshakeConfig),
		l:                l,
	}
	ifce.pki.cs.Store(cs)

	// Create manager
	conf := config.NewC(l)
	punchy := NewPunchyFromConfig(l, conf)
	nc := newConnectionManagerFromConfig(l, conf, hostMap, punchy)
	nc.intf = ifce
	p := []byte("")
	nb := make([]byte, 12, 12)
	out := make([]byte, mtu)

	// Add an ip we have established a connection w/ to hostmap
	hostinfo := &HostInfo{
		vpnAddrs:      []netip.Addr{vpnIp},
		localIndexId:  1099,
		remoteIndexId: 9901,
	}
	hostinfo.ConnectionState = &ConnectionState{
		myCert: &dummyCert{version: cert.Version1},
		H:      &noise.HandshakeState{},
	}
	nc.hostMap.unlockedAddHostInfo(hostinfo, ifce)

	// We saw traffic out to vpnIp
	nc.Out(hostinfo)
	nc.In(hostinfo)
	assert.True(t, hostinfo.in.Load())
	assert.True(t, hostinfo.out.Load())
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)

	// Do a traffic check tick, should not be pending deletion but should not have any in/out packets recorded
	nc.doTrafficCheck(hostinfo.localIndexId, p, nb, out, time.Now())
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())

	// Do another traffic check tick, this host should be pending deletion now
	nc.Out(hostinfo)
	nc.doTrafficCheck(hostinfo.localIndexId, p, nb, out, time.Now())
	assert.True(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])

	// We saw traffic, should no longer be pending deletion
	nc.In(hostinfo)
	nc.doTrafficCheck(hostinfo.localIndexId, p, nb, out, time.Now())
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])
}

func Test_NewConnectionManager_DisconnectInactive(t *testing.T) {
	l := test.NewLogger()
	localrange := netip.MustParsePrefix("10.1.1.1/24")
	vpnAddrs := []netip.Addr{netip.MustParseAddr("172.1.1.2")}
	preferredRanges := []netip.Prefix{localrange}

	// Very incomplete mock objects
	hostMap := newHostMap(l)
	hostMap.preferredRanges.Store(&preferredRanges)

	cs := &CertState{
		initiatingVersion: cert.Version1,
		privateKey:        []byte{},
		v1Cert:            &dummyCert{version: cert.Version1},
		v1HandshakeBytes:  []byte{},
	}

	lh := newTestLighthouse()
	ifce := &Interface{
		hostMap:          hostMap,
		inside:           &test.NoopTun{},
		outside:          &udp.NoopConn{},
		firewall:         &Firewall{},
		lightHouse:       lh,
		pki:              &PKI{},
		handshakeManager: NewHandshakeManager(l, hostMap, lh, &udp.NoopConn{}, defaultHandshakeConfig),
		l:                l,
	}
	ifce.pki.cs.Store(cs)

	// Create manager
	conf := config.NewC(l)
	conf.Settings["tunnels"] = map[string]any{
		"drop_inactive": true,
	}
	punchy := NewPunchyFromConfig(l, conf)
	nc := newConnectionManagerFromConfig(l, conf, hostMap, punchy)
	assert.True(t, nc.dropInactive.Load())
	nc.intf = ifce

	// Add an ip we have established a connection w/ to hostmap
	hostinfo := &HostInfo{
		vpnAddrs:      vpnAddrs,
		localIndexId:  1099,
		remoteIndexId: 9901,
	}
	hostinfo.ConnectionState = &ConnectionState{
		myCert: &dummyCert{version: cert.Version1},
		H:      &noise.HandshakeState{},
	}
	nc.hostMap.unlockedAddHostInfo(hostinfo, ifce)

	// Do a traffic check tick, in and out should be cleared but should not be pending deletion
	nc.Out(hostinfo)
	nc.In(hostinfo)
	assert.True(t, hostinfo.out.Load())
	assert.True(t, hostinfo.in.Load())

	now := time.Now()
	decision, _, _ := nc.makeTrafficDecision(hostinfo.localIndexId, now)
	assert.Equal(t, tryRehandshake, decision)
	assert.Equal(t, now, hostinfo.lastUsed)
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())

	decision, _, _ = nc.makeTrafficDecision(hostinfo.localIndexId, now.Add(time.Second*5))
	assert.Equal(t, doNothing, decision)
	assert.Equal(t, now, hostinfo.lastUsed)
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())

	// Do another traffic check tick, should still not be pending deletion
	decision, _, _ = nc.makeTrafficDecision(hostinfo.localIndexId, now.Add(time.Second*10))
	assert.Equal(t, doNothing, decision)
	assert.Equal(t, now, hostinfo.lastUsed)
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])

	// Finally advance beyond the inactivity timeout
	decision, _, _ = nc.makeTrafficDecision(hostinfo.localIndexId, now.Add(time.Minute*10))
	assert.Equal(t, closeTunnel, decision)
	assert.Equal(t, now, hostinfo.lastUsed)
	assert.False(t, hostinfo.pendingDeletion.Load())
	assert.False(t, hostinfo.out.Load())
	assert.False(t, hostinfo.in.Load())
	assert.Contains(t, nc.hostMap.Indexes, hostinfo.localIndexId)
	assert.Contains(t, nc.hostMap.Hosts, hostinfo.vpnAddrs[0])
}

// Check if we can disconnect the peer.
// Validate if the peer's certificate is invalid (expired, etc.)
// Disconnect only if disconnectInvalid: true is set.
func Test_NewConnectionManagerTest_DisconnectInvalid(t *testing.T) {
	now := time.Now()
	l := test.NewLogger()

	vpncidr := netip.MustParsePrefix("172.1.1.1/24")
	localrange := netip.MustParsePrefix("10.1.1.1/24")
	vpnIp := netip.MustParseAddr("172.1.1.2")
	preferredRanges := []netip.Prefix{localrange}
	hostMap := newHostMap(l)
	hostMap.preferredRanges.Store(&preferredRanges)

	// Generate keys for CA and peer's cert.
	pubCA, privCA, _ := ed25519.GenerateKey(rand.Reader)
	tbs := &cert.TBSCertificate{
		Version:   1,
		Name:      "ca",
		IsCA:      true,
		NotBefore: now,
		NotAfter:  now.Add(1 * time.Hour),
		PublicKey: pubCA,
	}

	caCert, err := tbs.Sign(nil, cert.Curve_CURVE25519, privCA)
	require.NoError(t, err)
	ncp := cert.NewCAPool()
	require.NoError(t, ncp.AddCA(caCert))

	pubCrt, _, _ := ed25519.GenerateKey(rand.Reader)
	tbs = &cert.TBSCertificate{
		Version:   1,
		Name:      "host",
		Networks:  []netip.Prefix{vpncidr},
		NotBefore: now,
		NotAfter:  now.Add(60 * time.Second),
		PublicKey: pubCrt,
	}
	peerCert, err := tbs.Sign(caCert, cert.Curve_CURVE25519, privCA)
	require.NoError(t, err)

	cachedPeerCert, err := ncp.VerifyCertificate(now.Add(time.Second), peerCert)

	cs := &CertState{
		privateKey:       []byte{},
		v1Cert:           &dummyCert{},
		v1HandshakeBytes: []byte{},
	}

	lh := newTestLighthouse()
	ifce := &Interface{
		hostMap:          hostMap,
		inside:           &test.NoopTun{},
		outside:          &udp.NoopConn{},
		firewall:         &Firewall{},
		lightHouse:       lh,
		handshakeManager: NewHandshakeManager(l, hostMap, lh, &udp.NoopConn{}, defaultHandshakeConfig),
		l:                l,
		pki:              &PKI{},
	}
	ifce.pki.cs.Store(cs)
	ifce.pki.caPool.Store(ncp)
	ifce.disconnectInvalid.Store(true)

	// Create manager
	conf := config.NewC(l)
	punchy := NewPunchyFromConfig(l, conf)
	nc := newConnectionManagerFromConfig(l, conf, hostMap, punchy)
	nc.intf = ifce
	ifce.connectionManager = nc

	hostinfo := &HostInfo{
		vpnAddrs: []netip.Addr{vpnIp},
		ConnectionState: &ConnectionState{
			myCert:   &dummyCert{},
			peerCert: cachedPeerCert,
			H:        &noise.HandshakeState{},
		},
	}
	nc.hostMap.unlockedAddHostInfo(hostinfo, ifce)

	// Move ahead 45s.
	// Check if to disconnect with invalid certificate.
	// Should be alive.
	nextTick := now.Add(45 * time.Second)
	invalid := nc.isInvalidCertificate(nextTick, hostinfo)
	assert.False(t, invalid)

	// Move ahead 61s.
	// Check if to disconnect with invalid certificate.
	// Should be disconnected.
	nextTick = now.Add(61 * time.Second)
	invalid = nc.isInvalidCertificate(nextTick, hostinfo)
	assert.True(t, invalid)
}

type dummyCert struct {
	version        cert.Version
	curve          cert.Curve
	groups         []string
	isCa           bool
	issuer         string
	name           string
	networks       []netip.Prefix
	notAfter       time.Time
	notBefore      time.Time
	publicKey      []byte
	signature      []byte
	unsafeNetworks []netip.Prefix
}

func (d *dummyCert) Version() cert.Version {
	return d.version
}

func (d *dummyCert) Curve() cert.Curve {
	return d.curve
}

func (d *dummyCert) Groups() []string {
	return d.groups
}

func (d *dummyCert) IsCA() bool {
	return d.isCa
}

func (d *dummyCert) Issuer() string {
	return d.issuer
}

func (d *dummyCert) Name() string {
	return d.name
}

func (d *dummyCert) Networks() []netip.Prefix {
	return d.networks
}

func (d *dummyCert) NotAfter() time.Time {
	return d.notAfter
}

func (d *dummyCert) NotBefore() time.Time {
	return d.notBefore
}

func (d *dummyCert) PublicKey() []byte {
	return d.publicKey
}

func (d *dummyCert) Signature() []byte {
	return d.signature
}

func (d *dummyCert) UnsafeNetworks() []netip.Prefix {
	return d.unsafeNetworks
}

func (d *dummyCert) MarshalForHandshakes() ([]byte, error) {
	return nil, nil
}

func (d *dummyCert) Sign(curve cert.Curve, key []byte) error {
	return nil
}

func (d *dummyCert) CheckSignature(key []byte) bool {
	return true
}

func (d *dummyCert) Expired(t time.Time) bool {
	return false
}

func (d *dummyCert) CheckRootConstraints(signer cert.Certificate) error {
	return nil
}

func (d *dummyCert) VerifyPrivateKey(curve cert.Curve, key []byte) error {
	return nil
}

func (d *dummyCert) String() string {
	return ""
}

func (d *dummyCert) Marshal() ([]byte, error) {
	return nil, nil
}

func (d *dummyCert) MarshalPEM() ([]byte, error) {
	return nil, nil
}

func (d *dummyCert) Fingerprint() (string, error) {
	return "", nil
}

func (d *dummyCert) MarshalJSON() ([]byte, error) {
	return nil, nil
}

func (d *dummyCert) Copy() cert.Certificate {
	return d
}
