const API = '/api';

document.addEventListener('DOMContentLoaded', () => {
  fetchChain();

  document.getElementById('create-form').onsubmit = e => handleSubmit(e, `${API}/auction/create`, fetchChain);
  document.getElementById('bid-form').onsubmit = e => handleSubmit(e, `${API}/auction/bid`, fetchChain);
  document.getElementById('close-form').onsubmit = e => handleSubmit(e, `${API}/auction/close`, fetchChain);

  document.getElementById('validate-btn').onclick = () => {
    fetch(`${API}/validate?t=${Date.now()}`)
      .then(r => r.json())
      .then(res => {
        if (res.valid) {
          alert('Цепь цела.');
          fetchChain();
        } else {
          setCompromisedState(true);
          alert('Цепь повреждена!');
        }
      })
      .catch(console.error);
  };
});

async function handleSubmit(e, endpoint, callback) {
  e.preventDefault();
  const fd = new FormData(e.target);
  const body = Object.fromEntries(fd);

  if (body.start_price) body.start_price = parseFloat(body.start_price);
  if (body.amount) body.amount = parseFloat(body.amount);

  try {
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body)
    });

    if (!res.ok) {
      let errData = {};
      try { errData = await res.json(); } catch {}
      
      const errMsg = errData.error || await res.text();

      if (errData.status === 'compromised') {
        setCompromisedState(true);
        throw new Error('Цепь повреждена! Редактирование заблокировано.');
      }
      throw new Error(errMsg || `HTTP ${res.status}`);
    }

    const result = await res.json();
    console.log('Ответ сервера:', result);
    e.target.reset();
    callback();
  } catch (err) {
    console.error('Ошибка:', err);
    alert(err.message);
  }
}

async function fetchChain() {
  try {
    const res = await fetch(`${API}/chain?t=${Date.now()}`);
    if (!res.ok) throw new Error(res.statusText);
    const chain = await res.json();

    const html = chain.map(r => `
      <div class="block">
        <b>#${r.index}</b> | ${r.type}<br>
        <span class="hash">${new Date(r.timestamp).toLocaleString()}</span><br>
        ${JSON.stringify(r.payload)}<br>
        <span class="hash">${r.hash.substring(0, 16)}...</span>
      </div>
    `).join('');

    document.getElementById('chain-list').innerHTML = html;
  } catch (err) {
    console.error('Не удалось загрузить цепь:', err);
  }
}

// 🔒 Управление состоянием "цепь взломана"
function setCompromisedState(isCompromised) {
  const banner = document.getElementById('status-banner');
  const buttons = document.querySelectorAll('form button');

  if (isCompromised) {
    banner.classList.remove('hidden');
    buttons.forEach(btn => btn.disabled = true);
  } else {
    banner.classList.add('hidden');
    buttons.forEach(btn => btn.disabled = false);
  }
}