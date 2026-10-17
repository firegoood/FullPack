import * as api from '../api.js';

export function wireAutoPortSuggestions(group, update, forwards = () => '') {
  let request = 0, edited = false;
  const field = key => group.querySelector(`[data-peer-field="${key}"]`);
  const status = group.querySelector('[data-peer-port-status]');
  for (const key of ['quick.basePort', 'quick.directPort']) {
    field(key).addEventListener('input', () => {
      edited = true; request++;
      status.textContent = 'Manual ports will be checked before saving.';
    });
  }
  const suggest = async (force = false) => {
    if (edited && !force) return;
    const current = ++request;
    status.textContent = 'Checking free local listener ports…';
    try {
      const ports = await api.suggestAutoPorts(field('quick.mode').value, forwards());
      if (current !== request) return;
      field('quick.basePort').value = String(ports.basePort);
      field('quick.directPort').value = String(ports.directPort);
      edited = false;
      status.textContent = 'Iran listener ports checked. Kharej verifies its own ports on import. Ports are rechecked before saving.';
      update();
    } catch (e) {
      if (current === request) status.textContent = e.message;
    }
  };
  group.querySelector('[data-peer-free-ports]').onclick = () => suggest(true);
  field('quick.mode').addEventListener('change', () => suggest());
  suggest();
}
