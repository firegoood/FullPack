import { test } from 'node:test';
import assert from 'node:assert/strict';

globalThis.document = { documentElement: { dataset: {} } };
const { peerSetupPayload, peerEditForm, peerSetupHost } = await import('../panel/js/ui/peerauto.js');
const { wireAutoPortSuggestions } = await import('../panel/js/ui/autoports.js');
const controls = values => ({ querySelector: selector => {
  const key = selector.match(/data-peer-field="([^"]+)"/)?.[1];
  const value = values[key];
  return value === undefined ? null : typeof value === 'boolean' ? { type: 'checkbox', checked: value } : { type: 'text', value: String(value) };
} });
test('Auto edit changes only local fields and keeps redacted secrets for backend preservation', () => {
  const a = {enabled:false,tunnelId:'paired',observeOnly:true,maxMemoryMb:256,mode:'l3',l3:{iranIp:'10.44.0.1/30',kharejIp:'10.44.0.2/30',mtu:1400,interface:'old'},candidates:[{id:'tls',transport:'reality',mode:'direct',endpoint:'peer.example:443',helper:{binary:'/old/xray',reversePort:3088,publicKey:'paired-public',privateKey:''}},{id:'raw',transport:'pck',endpoint:'192.0.2.1:8443'}]};
  const saved=JSON.stringify(a);
  const group=controls({'observeOnly':false,'maxMemoryMb':384,'local.tun':'fp-new','local.0.bind':'0.0.0.0:443','local.0.binary':'/custom/xray','local.0.privateKey':'','local.1.interface':'eth1'});
  const panes={innerHTML:'',querySelector:()=>group};
  const getEdit=peerEditForm({querySelector:()=>panes,querySelectorAll:()=>[]},{auto:a,role:'client'},'test');
  const result=getEdit().auto;
  assert.equal(result.candidates[0].localBind,'0.0.0.0:443'); assert.equal(result.candidates[0].helper.privateKey,''); assert.equal(result.candidates[0].helper.binary,'/custom/xray');
  assert.equal(result.candidates[1].packet.interface,'eth1'); assert.equal('localBind' in result.candidates[1],false);
  assert.equal(result.l3.interface,'fp-new');assert.equal(result.candidates[0].helper.reversePort,3088); assert.equal(result.candidates[0].endpoint,'peer.example:443'); assert.equal(result.candidates[0].helper.publicKey,'paired-public');
  assert.equal(JSON.stringify(a),saved);
});
test('native setup removes incompatible drawers and carries the explicit direction', () => {
  const group = controls({ acceptUdp: true, peerIp: '192.0.2.2', tunnelId: 'test-native-pair-identity', direction: 'reverse', maxPayload: 512, maxStreams: 128, paths: 4 });
  const result = peerSetupPayload({ querySelector: () => group }, { transport: 'waticmp', role: 'server', name: 'wat', token: 'secret', ports: '443', preset: 'turbo', tunnelPort: '8443', conn: { proxy: 'stale' }, managed: { privateKey: 'stale' } });
  assert.equal(result.waticmp.direction, 'reverse'); assert.equal(result.waticmp.peerIp, '192.0.2.2'); assert.equal(result.tune.acceptUDP, true);
  for (const key of ['preset', 'tunnelPort', 'conn', 'managed']) assert.equal(key in result, false, key);
});
test('Auto setup requires explicit approved endpoint data and conservative controls', () => {
  const row = controls({ id: 'noise', transport: 'stealth', endpoint: '192.0.2.1:24443', localBind: '0.0.0.0:24443' });
  const group = controls({ acceptUdp: false, tunnelId: 'pair-identity-example', iranId: 'iran-identity-example', kharejId: 'kharej-identity-example', observeOnly: true, probeInterval: 600, cooldown: 300, minimumDwell: 600, maxSwitchesPerHour: 2, connectionLimit: 0, maxMemoryMb: 256 });
  group.querySelectorAll = () => [row];
  const result = peerSetupPayload({ querySelector: () => group }, { transport: 'auto', role: 'server', name: 'auto', token: 'secret', ports: '443', ipv6: true });
  assert.equal(result.auto.enabled, true); assert.equal(result.auto.observeOnly, true);
  assert.equal(result.auto.connectionLimit, 0); assert.equal('maxConnections' in result.auto, false);
  assert.deepEqual(result.auto.candidates, [{ id: 'noise', transport: 'stealth', disabled: false, endpoint: '192.0.2.1:24443', localBind: '0.0.0.0:24443' }]);
  assert.equal(result.auto.cooldown, 300); assert.equal('ipv6' in result, false);
});
test('Kharej enrollment sends only the trusted link and local name', () => {
  const group = controls({ enrollmentLink: 'backpack://3.private' });
  const result = peerSetupPayload({ querySelector: () => group }, { name: 'peer', role: 'client', transport: 'auto', token: 'stale', ports: '443' });
  assert.deepEqual(result, { name: 'peer', role: 'client', transport: 'auto', peerEnrollment: 'backpack://3.private' });
});

test('Auto accepts an optional high cap and rejects invalid numbers without silently reverting to legacy default', () => {
  for (const value of [0, 4000]) {
    const group=controls({connectionLimit:value});group.querySelectorAll=()=>[];
    assert.equal(peerSetupPayload({querySelector:()=>group},{transport:'auto',role:'server'}).auto.connectionLimit,value);
  }
  for (const value of [-1, 1.5, 'invalid', 1e30]) {
    const group=controls({connectionLimit:value});group.querySelectorAll=()=>[];
    assert.throws(()=>peerSetupPayload({querySelector:()=>group},{transport:'auto',role:'server'}),/0 means no limit/);
  }
});

test('Quick Auto sends only the reviewed IP/port inputs and skips advanced enrollment fields', () => {
  const group = controls({setupMode:'quick', 'quick.iranIp':'192.0.2.1', 'quick.kharejIp':'192.0.2.2', 'quick.basePort':25000, 'quick.directPort':25003, acceptUdp:true, observeOnly:true, tunnelId:'stale', iranId:'stale', kharejId:'stale'});
  group.querySelectorAll = () => { throw new Error('quick setup must not read advanced candidates'); };
  const result = peerSetupPayload({querySelector:()=>group}, {name:'quick', role:'server', transport:'auto', token:'secret', ports:'2052,8880', managed:{privateKey:'stale'}, preset:'stale'});
  assert.deepEqual(result.quickAuto, {iranIp:'192.0.2.1', kharejIp:'192.0.2.2', basePort:25000, directPort:25003, mode:'stream'});
  assert.equal(result.tune.acceptUDP, true);
  for (const key of ['auto', 'managed', 'preset']) assert.equal(key in result,false);
});

test('Quick Auto shows consistent default ports when custom port fields are blank', () => {
  const group=controls({setupMode:'quick', 'quick.iranIp':'192.0.2.1', 'quick.kharejIp':'192.0.2.2', 'quick.basePort':'', 'quick.directPort':''});
  const result=peerSetupPayload({querySelector:()=>group}, {role:'server',transport:'auto'});
  assert.equal(result.quickAuto.basePort,24443); assert.equal(result.quickAuto.directPort,24475);
});

test('Quick Auto fills checked suggestions and a late response cannot overwrite manual ports', async () => {
  const previous = globalThis.fetch, pending = [], urls = [];
  globalThis.fetch = url => { urls.push(url); return new Promise(resolve => pending.push(value => resolve({ok:true,json:async()=>value}))); };
  const nodes = new Map(), field = key => `[data-peer-field="${key}"]`;
  for (const [key, value] of Object.entries({setupMode:'quick','quick.mode':'stream','quick.iranIp':'192.0.2.1','quick.kharejIp':'192.0.2.2','quick.basePort':'24443','quick.directPort':'24475'})) {
    nodes.set(field(key), {value, listeners:{}, addEventListener(event, fn) { (this.listeners[event] ||= []).push(fn); }});
  }
  for (const key of ['quick','advanced','quick-review','port-status','free-ports']) nodes.set(`[data-peer-${key}]`, {});
  const group = {querySelector: selector => nodes.get(selector)};
  try {
    wireAutoPortSuggestions(group, () => {}, () => '2052,8880');
    assert.match(urls[0], /what=auto-ports/); assert.match(urls[0], /ports=2052%2C8880/);
    const base = nodes.get(field('quick.basePort')); base.value = '27000';
    for (const fn of base.listeners.input) fn();
    pending.shift()({basePort:24699,directPort:24731});
    await new Promise(resolve => setImmediate(resolve));
    assert.equal(base.value,'27000');
    assert.match(nodes.get('[data-peer-port-status]').textContent,/Manual ports/);
    const refreshed = nodes.get('[data-peer-free-ports]').onclick();
    pending.shift()({basePort:24955,directPort:24987}); await refreshed;
    assert.equal(base.value,'24955'); assert.equal(nodes.get(field('quick.directPort')).value,'24987');
    assert.match(nodes.get('[data-peer-port-status]').textContent,/Kharej verifies/);
  } finally { globalThis.fetch = previous; }
});

test('Quick Auto L3 requests the separate datagram pool without advanced hidden fields', () => {
  const group=controls({setupMode:'quick', 'quick.mode':'l3', 'quick.iranIp':'192.0.2.1', 'quick.kharejIp':'192.0.2.2'});
  const result=peerSetupPayload({querySelector:()=>group}, {role:'server',transport:'auto',ports:''});
  assert.equal(result.quickAuto.mode,'l3'); assert.equal(result.ports,''); assert.equal('auto' in result,false);
});

test('peer setup links use the visible form after switching between Quick Auto and Wat', () => {
  const auto = controls({setupMode:'quick', 'quick.iranIp':'192.0.2.1'});
  const wat = controls({iranIp:'192.0.2.3'});
  const root = {querySelector: selector => selector.includes('"auto"') ? auto : wat};
  auto.hidden = false; wat.hidden = true;
  assert.equal(peerSetupHost(root), '192.0.2.1');
  auto.hidden = true; wat.hidden = false;
  assert.equal(peerSetupHost(root), '192.0.2.3');
  auto.hidden = false; wat.hidden = true;
  auto.querySelector = controls({setupMode:'advanced'}).querySelector;
  assert.equal(peerSetupHost(root), '');
  auto.hidden = true;
  assert.equal(peerSetupHost(root), '');
});

test('Auto Direct HTTPS excludes listener keys and incompatible hidden fields', () => {
  const row = controls({ id: 'https', transport: 'xhttp', mode: 'direct', endpoint: 'peer.example:443', localBind: 'stale:443', 'helper.reversePort': 3088, 'helper.binary': '/opt/xray', 'helper.serverName': 'peer.example', 'helper.uuid': '12345678-1234-1234-1234-123456789abc', 'helper.path': '/private', 'helper.host': '', 'helper.caFile': '/opt/ca.pem', 'helper.caPem': 'public trust', 'helper.privateKey': 'hidden private', 'helper.password': 'hidden password', 'helper.certificate': '/secret/server.crt', 'packet.iranSource': 'stale' });
  const group = controls({}); group.querySelectorAll = () => [row];
  const c = peerSetupPayload({ querySelector: () => group }, {transport: 'auto', role: 'server'}).auto.candidates[0];
  assert.equal(c.mode, 'direct'); assert.equal(c.helper.caPem, 'public trust');
  for (const key of ['certificate', 'privateKey', 'password']) assert.equal(key in c.helper, false);
  assert.equal('localBind' in c, false); assert.equal('packet' in c, false);
});

test('Auto L3 carries shared TUN addresses and only the selected raw profile', () => {
  const row = controls({ id: 'raw', transport: 'spoof', mode: 'direct', endpoint: '192.0.2.2:8000', spoofIranIp: '192.0.2.1', spoofKharejIp: '192.0.2.2', 'packet.iranSource': '198.51.100.1', 'packet.kharejSource': '198.51.100.2', 'packet.spoofProfile': 'udp', 'packet.interface': 'eth1', 'packet.sniDomain': 'stale', 'helper.uuid': 'stale' });
  const group = controls({autoMode: 'l3', 'l3.iranIp': '10.44.0.1/30', 'l3.kharejIp': '10.44.0.2/30', 'l3.mtu': 1400, 'l3.interface': 'fp0'}); group.querySelectorAll = () => [row];
  const a = peerSetupPayload({querySelector: () => group}, {transport: 'auto', role: 'server', ports: ''}).auto;
  assert.equal(a.mode, 'l3'); assert.deepEqual(a.l3, {iranIp:'10.44.0.1/30', kharejIp:'10.44.0.2/30', mtu:1400, interface:'fp0'});
  assert.equal(a.candidates[0].packet.iranSource, '198.51.100.1'); assert.equal('sniDomain' in a.candidates[0].packet, false); assert.equal('helper' in a.candidates[0], false);
});
