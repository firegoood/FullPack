import { esc } from '../lib/dom.js';
import * as api from '../api.js';

const isPeerTransport = t => t === 'waticmp' || t === 'auto';
const forms = new WeakMap();
const field = (key, title, value = '', type = 'text', hint = '') =>
  `<label class="f3"><span>${esc(title)}</span>${type === 'textarea' ? `<textarea data-peer-field="${esc(key)}">${esc(String(value))}</textarea>` : `<input data-peer-field="${esc(key)}" type="${type}" ${type === 'checkbox' ? (value ? 'checked' : '') : `value="${esc(String(value))}"`}>`}<small class="hint">${esc(hint)}</small></label>`;
const direction = value => `<label class="f3"><span>Echo Request direction</span><select data-peer-field="direction"><option value="normal" ${value === 'normal' ? 'selected' : ''}>Normal: Iran → Kharej</option><option value="reverse" ${value === 'reverse' ? 'selected' : ''}>Reverse: Kharej → Iran</option></select></label>`;
const newID = () => Array.from(crypto.getRandomValues(new Uint8Array(16)), b => b.toString(16).padStart(2, '0')).join('');
const get = (root, key) => {
  const n = root?.querySelector(`[data-peer-field="${key}"]`);
  return n?.type === 'checkbox' ? n.checked : n?.value.trim() || '';
};
const num = (root, key) => Number(get(root, key));
const watFields = (w = {}) => field('peerIp', 'Other server IPv4', w.peerIp || '', 'text', 'Explicit address; ICMP has no tunnel port.')
  + field('tunnelId', 'Shared tunnel ID', w.tunnelId || newID()) + direction(w.direction || 'normal')
  + field('maxPayload', 'Maximum native payload', w.maxPayload || 512, 'number', '256–1400 bytes; authenticated PMTU discovery may lower this.')
  + field('maxStreams', 'Maximum concurrent streams', w.maxStreams || 128, 'number')
  + field('paths', 'ICMP lanes', w.paths || 4, 'number');

function candidateMarkup(p = {}, index = 0, registry = []) {
  const h = p.helper || {}, packet = p.packet || {};
  return `<fieldset class="peer-candidate"><legend>Approved carrier ${index + 1}</legend>
    ${field('id', 'Candidate ID', p.id || `c${newID().slice(0, 8)}`)}
    <label class="f3"><span>Carrier</span><select data-peer-field="transport">${registry.map(c => `<option value="${esc(c.value)}" ${p.transport === c.value ? 'selected' : ''}>${esc(c.label)}</option>`).join('')}</select></label>
    <label class="f3"><span>Connection direction</span><select data-peer-field="mode"><option value="reverse">Reverse: Kharej connects to Iran</option><option value="direct" ${p.mode === 'direct' ? 'selected' : ''}>Direct: Iran connects to Kharej</option></select></label>
    <div data-peer-wat>${field('iranIp', 'Iran IPv4', p.iranIp || '')}${field('kharejIp', 'Kharej IPv4', p.kharejIp || '')}${direction(p.direction || 'normal')}${field('maxPayload', 'Maximum payload', p.maxPayload || 512, 'number')}</div>
    <div data-peer-endpoint>${field('endpoint', 'Approved listener host:port', p.endpoint || '', 'text', 'Iran for Reverse, Kharej for Direct. Explicitly permit this endpoint.')}<div data-peer-bind>${field('localBind', 'Local Iran listener bind (optional)', p.localBind || '', 'text', 'For example 0.0.0.0 with the same enrolled port.')}</div></div>
    <div data-peer-packet>${field('packet.interface', 'Local raw interface (optional)', packet.interface || '')}
      <div data-peer-spoof>${field('spoofIranIp', 'Iran real IPv4', p.iranIp || '')}${field('spoofKharejIp', 'Kharej real IPv4', p.kharejIp || '')}${field('packet.spoofProfile', 'Spoof profile', packet.spoofProfile || 'udp')}${field('packet.iranSource', 'Iran forged source IPv4', packet.iranSource || '')}${field('packet.kharejSource', 'Kharej forged source IPv4', packet.kharejSource || '')}</div>
      <div data-peer-sni>${field('packet.sniDomain', 'SNI domain', packet.sniDomain || 'www.microsoft.com')}</div></div>
    <div data-peer-helper>${field('helper.binary', 'Local helper executable (optional)', h.binary || '', 'text', 'Blank uses the installed role-specific helper; no automatic download.')}${field('helper.reversePort', 'Private loopback target port', h.reversePort || 3080 + index, 'number', 'Unique on the listener. This target is restricted to this tunnel.')}${field('helper.serverName', 'TLS / REALITY server name', h.serverName || '')}
      <div data-peer-naive>${field('helper.username', 'HTTPS account', h.username || 'fullpack')}${field('helper.password', 'HTTPS password', h.password || '', 'password', 'Blank on an Iran listener generates a secret; Direct needs the enrolled Kharej account.')}</div>
      <div data-peer-xray>${field('helper.uuid', 'Shared UUID', h.uuid || '', 'text', 'Blank on an Iran listener generates an identity.')}</div>
      <div data-peer-xhttp>${field('helper.path', 'Private XHTTP path', h.path || '')}${field('helper.host', 'XHTTP host override (optional)', h.host || '')}</div>
      <div data-peer-reality>${field('helper.target', 'REALITY cover host:port', h.target || '')}${field('helper.shortId', 'REALITY short ID', h.shortId || '')}<div data-peer-private>${field('helper.privateKey', 'Local REALITY private key (optional)', h.privateKey || '', 'password')}</div><div data-peer-public>${field('helper.publicKey', 'Kharej REALITY public key', h.publicKey || '')}</div></div>
      <div data-peer-certificate><div data-peer-cert-local>${field('helper.certificate', 'Local certificate path (optional)', h.certificate || '')}${field('helper.key', 'Local certificate key path (optional)', h.key || '')}</div><div data-peer-trust>${field('helper.caFile', 'Local CA file (optional)', h.caFile || '')}${field('helper.caPem', 'Enrolled public CA / certificate PEM (optional)', h.caPem || '', 'textarea')}</div></div></div>
    <button type="button" data-peer-remove>Remove carrier</button></fieldset>`;
}
function wireCandidates(group, registry = []) {
  group.querySelectorAll('.peer-candidate').forEach(row => {
    const select = row.querySelector('[data-peer-field="transport"]');
    const mode = row.querySelector('[data-peer-field="mode"]');
    const shape = () => {
      const c = registry.find(c => c.value === select.value) || {}, listener = mode.value !== 'direct', tr = select.value;
      const show = (key, yes) => { row.querySelector(`[data-peer-${key}]`).hidden = !yes; };
      show('wat', tr === 'waticmp'); show('endpoint', tr !== 'waticmp'); show('bind', listener);
      show('packet', c.raw && !['waticmp', 'xdi'].includes(tr)); show('spoof', tr === 'spoof'); show('sni', tr === 'sni');
      show('helper', c.helper); show('naive', tr === 'naive'); show('xray', tr === 'xhttp' || tr === 'reality'); show('xhttp', tr === 'xhttp'); show('reality', tr === 'reality');
      show('private', listener); show('public', !listener); show('certificate', tr !== 'reality'); show('cert-local', listener); show('trust', !listener);
    };
    if (!row.dataset.wired) { select.addEventListener('change', shape); mode.addEventListener('change', shape); row.dataset.wired = '1'; }
    shape();
    row.querySelector('[data-peer-remove]').onclick = () => row.remove();
  });
}
function autoFields(a = {}, registry = []) {
  return `<label class="f3"><span>Auto traffic mode</span><select data-peer-field="autoMode"><option value="stream">TCP/UDP port forwarding</option><option value="l3" ${a.mode === 'l3' ? 'selected' : ''}>Layer 3: stable TUN, datagram carriers</option></select></label><div data-peer-l3>${field('l3.iranIp', 'Iran TUN address/prefix', a.l3?.iranIp || '10.44.0.1/30')}${field('l3.kharejIp', 'Kharej TUN address/prefix', a.l3?.kharejIp || '10.44.0.2/30')}${field('l3.mtu', 'TUN MTU', a.l3?.mtu || 1400, 'number')}${field('l3.interface', 'Local TUN interface (optional)', a.l3?.interface || '')}<p>Forwarded ports are optional. Only enrolled datagram carriers carry routed IP.</p></div>` + field('tunnelId', 'Pair tunnel ID', a.tunnelId || newID())
    + field('iranId', 'Iran identity', a.iranId || newID()) + field('kharejId', 'Kharej identity', a.kharejId || newID())
    + field('observeOnly', 'Observe only', a.observeOnly ?? true, 'checkbox', 'Bootstrap the first approved carrier; automatic cross-carrier selection stays off.')
    + field('allowDegradationSwitch', 'Switch on sustained severe degradation', a.allowDegradationSwitch || false, 'checkbox')
    + field('probeInterval', 'Full probe interval (seconds)', a.probeInterval || 600, 'number')
    + field('cooldown', 'Optional switch cooldown (seconds)', a.cooldown || 300, 'number')
    + field('minimumDwell', 'Minimum dwell (seconds)', a.minimumDwell || 600, 'number')
    + field('maxSwitchesPerHour', 'Optional switches per hour', a.maxSwitchesPerHour || 2, 'number', 'Confirmed outages use a separate recovery gate: at least 5 seconds between changes, at most 12 committed changes per hour. Locks and observe-only still apply.')
    + field('maxConnections', 'Maximum concurrent application connections', a.maxConnections || 64, 'number', 'Default 64, maximum 192. Connections are not a measured user capacity.')
    + field('maxMemoryMb', 'Probe memory pressure threshold (MB)', a.maxMemoryMb || 256, 'number')
    + `<div data-peer-candidates>${(a.candidates?.length ? a.candidates : [{}]).map((p, i) => candidateMarkup(p, i, registry)).join('')}</div><button type="button" data-peer-add>Add approved carrier (up to four)</button>`;
}
export function resetPeerSetup(root) {
  root.querySelectorAll('[data-peer-hidden]').forEach(n => { n.hidden = false; delete n.dataset.peerHidden; });
  root.querySelectorAll('.peer-settings').forEach(n => { n.hidden = true; });
}
export function shapePeerSetup(root, transport, role, direct = false, registry = []) {
  const enabled = !direct && isPeerTransport(transport);
  resetPeerSetup(root);
  const here = root.querySelector('.here-card');
  if (here && enabled) {
    here.querySelector('.here-flag').textContent = role === 'client' ? '🌍' : '🇮🇷';
    here.querySelector('b').textContent = role === 'client' ? 'Kharej — this server' : 'Iran — this server';
    here.querySelector('small').textContent = role === 'client'
      ? 'Import the trusted Iran enrollment link here. Both servers need this feature version.'
      : 'Build Iran here, then transfer its private setup link to Kharej.';
  }
  if (!enabled) return;
  root.querySelectorAll('input[name], select[name]').forEach(n => {
    if (['name', ...(role === 'server' ? ['token', 'ports'] : [])].includes(n.name)) return;
    const row = n.closest('.f3, .tg3, .dr2b, .grp3');
    if (row && !row.hidden) { row.hidden = true; row.dataset.peerHidden = '1'; }
  });
  root.querySelectorAll('.dr2b, .rp').forEach(n => { if (!n.hidden) { n.hidden = true; n.dataset.peerHidden = '1'; } });
  // These groups contain only legacy presets/drawers. Hide the whole group so
  // the wizard can skip an empty step, while reset restores other transports.
  root.querySelectorAll('.rpgrid, .dr2b').forEach(n => {
    const group = n.closest('.grp3');
    if (group && !group.hidden) { group.hidden = true; group.dataset.peerHidden = '1'; }
  });
  let cache = forms.get(root); if (!cache) { cache = {}; forms.set(root, cache); }
  const cacheKey = `${transport}-${role}`;
  let group = cache[cacheKey];
  if (!group) {
    group = document.createElement('div'); group.className = 'grp3 peer-settings';
    group.dataset.peerTransport = transport;
    group.dataset.peerRole = role;
    group.innerHTML = `<h3>${transport === 'auto' ? 'Peer Auto (experimental)' : 'Wat ICMP (WatTunnel Engine)'}</h3><p>Linux is required. Wat needs CAP_NET_RAW. TCP/UDP forwarding stays on Iran. Keep the setup link private; it contains the enrollment secret.</p>`
      + `<button type="button" data-peer-side>${role === 'client' ? 'Build Iran instead' : 'Import an enrolled Kharej endpoint'}</button>`
      + (role === 'client' ? field('enrollmentLink', 'Trusted Iran enrollment link', '', 'password', 'Version 3 only. Match this carrier and obtain the secret-bearing link over a trusted channel; both peers need the new version.')
        + '<div data-peer-local-settings></div><p data-peer-preview-status role="status"></p>'
        : field('acceptUdp', 'Forward UDP as well as TCP', false, 'checkbox')
        + (transport === 'auto' ? autoFields({}, registry) : field('iranIp', 'This Iran server IPv4', '', 'text', 'Included in the peer setup link.') + watFields()));
    const anchor = root.querySelector('[name="ports"]')?.closest('.grp3') || root.querySelector('[name="name"]')?.closest('.grp3');
    anchor?.after(group); cache[cacheKey] = group;
    group.querySelector('[data-peer-side]').onclick = () => {
      root.querySelector(`[data-fn="setSide"][data-args*="${role === 'client' ? 'server' : 'client'}"]`)?.click();
    };
    if (transport === 'auto' && role === 'server') {
      wireCandidates(group, registry);
      const reshapeMode = () => {
        const l3 = get(group, 'autoMode') === 'l3'; group.querySelector('[data-peer-l3]').hidden = !l3;
        group.querySelectorAll('.peer-candidate [data-peer-field="transport"]').forEach(select => { const current = select.value; select.innerHTML = registry.filter(c => !l3 || c.l3).map(c => `<option value="${esc(c.value)}">${esc(c.label)}</option>`).join(''); if ([...select.options].some(o => o.value === current)) select.value = current; });
        wireCandidates(group, registry);
      };
      group.querySelector('[data-peer-field="autoMode"]').addEventListener('change', reshapeMode); reshapeMode();
      group.querySelector('[data-peer-add]').onclick = () => {
        const box = group.querySelector('[data-peer-candidates]');
        if (box.children.length >= 4) return;
        box.insertAdjacentHTML('beforeend', candidateMarkup({}, box.children.length, registry)); reshapeMode();
      };
    }
    if (role === 'client') wireLocalEnrollment(group, transport);
  }
  group.hidden = false;
}

function nativePayload(group) {
  return { enabled: true, peerIp: get(group, 'peerIp'), tunnelId: get(group, 'tunnelId'), direction: get(group, 'direction'),
    maxPayload: num(group, 'maxPayload'), maxStreams: num(group, 'maxStreams'), paths: num(group, 'paths') };
}

function wireLocalEnrollment(group, transport) {
  const input = group.querySelector('[data-peer-field="enrollmentLink"]'), box = group.querySelector('[data-peer-local-settings]'), status = group.querySelector('[data-peer-preview-status]');
  let revision = 0, timer;
  input.addEventListener('input', () => {
    const mine = ++revision; clearTimeout(timer); box.innerHTML = ''; status.textContent = '';
    timer = setTimeout(async () => {
      if (!input.value.trim()) return;
      try {
        const result = await api.peerEnrollmentPreview(transport, input.value.trim()); if (revision !== mine) return;
        box.innerHTML = (result.l3 ? `<div data-peer-local="">${field('interface', 'Local TUN interface (optional)')}</div>` : '') + result.candidates.map(c => `<fieldset data-peer-local="${esc(c.id)}"><legend>${esc(c.id)} · ${esc(c.transport)} · local settings</legend>`
          + (c.listener ? field('localBind', 'Kharej listener bind (optional)', c.defaultBind || '') : '')
          + (c.raw ? field('interface', 'Local raw interface (optional)') : '')
          + (c.helper ? field('binary', 'Local helper executable (optional)', '', 'text', 'Blank uses the installed role-specific helper.')
            + (c.listener ? c.transport === 'reality' ? field('privateKey', 'Matching local REALITY private key', '', 'password') : field('certificate', 'Matching local certificate path') + field('key', 'Local certificate key path') : field('caFile', 'Local CA file (optional)')) : '') + '</fieldset>').join('');
        status.textContent = 'Enrollment checked. Complete any required local helper identity before creating this peer.';
      } catch (e) { if (revision === mine) status.textContent = e.message; }
    }, 400);
  });
}

function autoCandidatePayload(row) {
  const c = { id: get(row, 'id'), transport: get(row, 'transport'), disabled: false };
  if (get(row, 'mode') === 'direct') c.mode = 'direct';
  if (c.transport === 'waticmp') Object.assign(c, { iranIp: get(row, 'iranIp'), kharejIp: get(row, 'kharejIp'), direction: get(row, 'direction'), maxPayload: num(row, 'maxPayload') });
  else { Object.assign(c, { endpoint: get(row, 'endpoint') }); if (!c.mode) c.localBind = get(row, 'localBind'); }
  if (['pck', 'spoof', 'sni'].includes(c.transport)) {
    c.packet = { interface: get(row, 'packet.interface') };
    if (c.transport === 'sni') c.packet.sniDomain = get(row, 'packet.sniDomain');
    if (c.transport === 'spoof') { Object.assign(c, {iranIp: get(row, 'spoofIranIp'), kharejIp: get(row, 'spoofKharejIp')}); Object.assign(c.packet, {spoofProfile: get(row, 'packet.spoofProfile'), iranSource: get(row, 'packet.iranSource'), kharejSource: get(row, 'packet.kharejSource')}); }
  }
  if (['naive', 'xhttp', 'reality'].includes(c.transport)) {
    c.helper = {reversePort: num(row, 'helper.reversePort'), binary: get(row, 'helper.binary'), serverName: get(row, 'helper.serverName')};
    const keys = c.transport === 'naive' ? ['username', 'password'] : c.transport === 'xhttp' ? ['uuid', 'path', 'host'] : ['uuid', 'target', 'shortId', c.mode ? 'publicKey' : 'privateKey'];
    keys.push(...(c.transport !== 'reality' ? c.mode ? ['caFile', 'caPem'] : ['certificate', 'key'] : []));
    for (const key of keys) c.helper[key] = get(row, `helper.${key}`);
  }
  return c;
}
// Retain only supported setup fields. Hidden legacy drawers never enter the API.
export function peerSetupPayload(root, payload) {
  if (!isPeerTransport(payload.transport)) return payload;
  const group = root.querySelector(`.peer-settings[data-peer-transport="${payload.transport}"][data-peer-role="${payload.role}"]`);
  if (payload.role === 'client') {
    const result = { name: payload.name, role: payload.role, transport: payload.transport, peerEnrollment: get(group, 'enrollmentLink') };
    const local = [...(group.querySelectorAll?.('[data-peer-local]') || [])].map(row => { const v = {id: row.dataset.peerLocal}; for (const key of ['localBind', 'interface', 'binary', 'caFile', 'certificate', 'key', 'privateKey']) if (get(row, key)) v[key] = get(row, key); return v; }).filter(v => Object.keys(v).length > 1);
    if (local.length) result.peerLocal = local;
    return result;
  }
  const result = { name: payload.name, role: payload.role, transport: payload.transport, token: payload.token, ports: payload.ports, tune: { acceptUDP: get(group, 'acceptUdp') } };
  if (payload.transport === 'waticmp') result.waticmp = nativePayload(group);
  else {
    const candidates = [...group.querySelectorAll('.peer-candidate')].map(autoCandidatePayload);
    result.auto = { enabled: true, tunnelId: get(group, 'tunnelId'), iranId: get(group, 'iranId'), kharejId: get(group, 'kharejId'),
      observeOnly: get(group, 'observeOnly'), allowDegradationSwitch: get(group, 'allowDegradationSwitch'), acceptUdp: get(group, 'acceptUdp'), candidates,
      probeInterval: num(group, 'probeInterval'), cooldown: num(group, 'cooldown'), minimumDwell: num(group, 'minimumDwell'), maxSwitchesPerHour: num(group, 'maxSwitchesPerHour'),
      maxConnections: num(group, 'maxConnections'), maxMemoryMb: num(group, 'maxMemoryMb') };
    if (get(group, 'autoMode') === 'l3') { result.auto.mode = 'l3'; result.auto.l3 = {iranIp: get(group, 'l3.iranIp'), kharejIp: get(group, 'l3.kharejIp'), mtu: num(group, 'l3.mtu'), interface: get(group, 'l3.interface')}; }
  }
  return result;
}
export const peerSetupHost = root => get(root.querySelector('.peer-settings[data-peer-transport="waticmp"]'), 'iranIp');

function localAutoFields(a, role) {
  return (a.l3 ? field('local.tun', 'Local TUN interface (blank = pair default)', a.l3.interface || '') : '')
    + a.candidates.map((p, i) => {
      const listener = (role === 'server') === (p.mode !== 'direct'), h = p.helper || {};
      return `<fieldset><legend>${esc(p.id)} · ${esc(p.transport)} · local settings</legend>`
        + (listener && p.transport !== 'waticmp' ? field(`local.${i}.bind`, 'Local listener bind (same enrolled port)', p.localBind || '') : '')
        + (['pck', 'spoof', 'sni'].includes(p.transport) ? field(`local.${i}.interface`, 'Local raw interface (optional)', p.packet?.interface || '') : '')
        + (p.helper ? field(`local.${i}.binary`, 'Local helper executable', h.binary || '')
          + (listener ? p.transport === 'reality' ? field(`local.${i}.privateKey`, 'Local REALITY private key', '', 'password', 'Blank keeps the current key. A replacement must match the enrolled public key.')
            : field(`local.${i}.certificate`, 'Matching local certificate path', h.certificate || '') + field(`local.${i}.key`, 'Local certificate key path', h.key || '')
          : p.transport !== 'reality' ? field(`local.${i}.caFile`, 'Local CA file (optional)', h.caFile || '') : '') : '') + '</fieldset>';
    }).join('');
}
function readLocalAuto(group, a, role) {
  const result = { ...a, observeOnly: get(group, 'observeOnly'), maxMemoryMb: num(group, 'maxMemoryMb'), candidates: a.candidates.map(p => ({...p, ...(p.packet ? {packet: {...p.packet}} : {}), ...(p.helper ? {helper: {...p.helper}} : {})})) };
  if (a.l3) result.l3 = {...a.l3, interface: get(group, 'local.tun')};
  result.candidates.forEach((p, i) => {
    const listener = (role === 'server') === (p.mode !== 'direct');
    if (listener && p.transport !== 'waticmp') p.localBind = get(group, `local.${i}.bind`);
    if (['pck', 'spoof', 'sni'].includes(p.transport)) p.packet = {...p.packet, interface: get(group, `local.${i}.interface`)};
    if (p.helper) {
      p.helper.binary = get(group, `local.${i}.binary`);
      if (listener) {
        if (p.transport === 'reality') p.helper.privateKey = get(group, `local.${i}.privateKey`);
        else { p.helper.certificate = get(group, `local.${i}.certificate`); p.helper.key = get(group, `local.${i}.key`); }
      } else if (p.transport !== 'reality') p.helper.caFile = get(group, `local.${i}.caFile`);
    }
  });
  return result;
}

export function peerEditForm(root, settings, name) {
  const panes = root.querySelector('.panes');
  root.querySelectorAll('.tabs > button:not(.hist)').forEach(n => { n.hidden = true; });
  const auto = settings.auto?.tunnelId;
  panes.innerHTML = `<div class="pane grp3 peer-edit"><h3>${auto ? 'Enrolled peer Auto' : 'Native Wat ICMP'}</h3>`
    + (auto ? `<p>Pair identities, backend maps and permitted endpoints are fixed by enrollment. Create a new pair to rotate its secret or change that manifest. Live actions below do not rewrite the config.</p>
        ${field('observeOnly', 'Observe only after restart', settings.auto.observeOnly, 'checkbox')}${field('maxMemoryMb', 'Probe memory pressure threshold (MB)', settings.auto.maxMemoryMb, 'number')}${localAutoFields(settings.auto, settings.role)}
        <div data-peer-status role="status"></div><button type="button" data-peer-refresh>Refresh status</button><div data-peer-actions></div>`
      : `<p>Direction and shared tunnel ID must match on both servers. An edit requires applying the new setup link on the peer.</p>${watFields(settings.waticmp)}`) + '</div>';
  const group = panes.querySelector('.peer-edit');
  if (auto && settings.auto.enabled) {
    const output = group.querySelector('[data-peer-status]');
    const actions = group.querySelector('[data-peer-actions]');
    const paint = s => {
      output.innerHTML = `<p><b>${esc(s.state)}</b> · Active: ${esc(s.active || 'none')} · Last good: ${esc(s.lastGood || 'none')} · Generation: ${esc(String(s.generation))}</p>
        <p>${esc(s.reason || '')}</p><p>Observe: ${s.observeOnly ? 'on' : 'off'} · Lock: ${esc(s.manualLock || 'none')}</p>
        <p>Last commit: ${esc(s.lastSwitch || 'none')} · Next eligible switch: ${esc(s.nextEligibleSwitch || 'policy permits')}</p>
        <p>Flows: ${Number(s.flows?.live || 0)} live, ${Number(s.flows?.drained || 0)} drained, ${Number(s.flows?.interrupted || 0)} interrupted, ${Number(s.flows?.rejected || 0)} refused; UDP queue drops: ${Number(s.flows?.udpQueueDrops || 0)}.</p>
        ${s.packets ? `<p>TUN ${esc(s.packets.interface || '')}: ${Number(s.packets.sent || 0)} IP packets sent, ${Number(s.packets.received || 0)} received, ${Number(s.packets.dropped || 0)} dropped.</p>` : ''}
        <table><thead><tr><th>Carrier / direction</th><th>Readiness</th><th>Probe Mbps / RTT ms</th></tr></thead><tbody>${(s.candidates || []).map(c => `<tr><td>${esc(c.id)} · ${esc(c.transport)} ${esc(c.mode || 'reverse')} ${esc(c.direction || '')}</td><td>${esc(c.ready ? 'ready' : c.reason || 'unavailable')}</td><td>${c.lastProbe && !c.lastProbe.startsWith('0001-') ? `${Number(c.usefulMbps || 0).toFixed(3)} / ${Number(c.rttMs || 0).toFixed(1)}` : 'not measured here'}</td></tr>`).join('')}</tbody></table>
        <small>Probe application traffic is capped at 128 KB/s. Reported throughput is a bounded lower-bound measurement.</small>
        <ul>${(s.history || []).slice(-8).map(e => `<li>${esc(e.at)} ${esc(e.candidate || '')} ${esc(e.result)}: ${esc(e.reason)}</li>`).join('')}</ul>`;
      actions.innerHTML = '';
      if (settings.role !== 'server') { actions.textContent = 'Switching actions are available on the enrolled Iran authority.'; return; }
      for (const [op, label] of [['observe', 'Observe only'], ['resume', 'Resume Auto'], ['unlock', 'Release runtime lock']]) addButton(op, '', label);
      for (const c of s.candidates || []) {
        for (const [op, label] of [['test-now', 'Test'], ['switch', 'Select'], ['lock', 'Lock'], ['disable-carrier', 'Disable until restart']]) addButton(op, c.id, `${label} ${c.id}`);
      }
    };
    const addButton = (op, candidate, label) => {
      const b = document.createElement('button'); b.type = 'button'; b.textContent = label; actions.append(b);
      b.onclick = async () => {
        actions.querySelectorAll('button').forEach(x => { x.disabled = true; }); output.textContent = 'Waiting for the peer-coordinated action…';
        try { paint(await api.autoAction(name, op, candidate)); }
        catch (e) { output.textContent = e.message; actions.querySelectorAll('button').forEach(x => { x.disabled = false; }); }
      };
    };
    const refresh = async () => { try { paint(await api.autoStatus(name)); } catch (e) { output.textContent = e.message; } };
    group.querySelector('[data-peer-refresh]').onclick = refresh; refresh();
  }
  return () => auto ? { auto: readLocalAuto(group, settings.auto, settings.role) }
    : { waticmp: { ...settings.waticmp, ...nativePayload(group) } };
}
