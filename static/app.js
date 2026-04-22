const API = '/api';

document.addEventListener('DOMContentLoaded', () => {
  fetchChain();

  // Привязка обработчиков
  document.getElementById('create-form').onsubmit = e => handleSubmit(e, `${API}/auction/create`, fetchChain);
  document.getElementById('bid-form').onsubmit = e => handleSubmit(e, `${API}/auction/bid`, fetchChain);
  document.getElementById('close-form').onsubmit = e => handleSubmit(e, `${API}/auction/close`, fetchChain);
  
  document.getElementById('hack-form').onsubmit = e => {
    e.preventDefault();
    const fd = new FormData(e.target);
    const body = Object.fromEntries(fd);
    body.index = parseInt(body.index);
    body.new_price = parseFloat(body.new_price);

    fetch(`${API}/hack/simulate`, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(body)
    }).then(r => r.json()).then(res => showHackResult(res))
      .catch(err => alert('❌ Ошибка запроса: ' + err.message));
  };

  document.getElementById('validate-btn').onclick = () => {
    fetch(`${API}/validate`).then(r => r.json()).then(res => {
      alert(res.valid ? '✅ Цепь цела.' : '❌ Цепь повреждена!');
    }).catch(console.error);
  };
});

async function handleSubmit(e, endpoint, callback) {
  e.preventDefault();
  const fd = new FormData(e.target);
  const body = Object.fromEntries(fd);

  // Конвертация чисел
  if (body.start_price) body.start_price = parseFloat(body.start_price);
  if (body.amount) body.amount = parseFloat(body.amount);

  try {
    console.log('📤 Отправка на:', endpoint, body);
    const res = await fetch(endpoint, {
      method: 'POST',
      headers: {'Content-Type': 'application/json'},
      body: JSON.stringify(body)
    });

    if (!res.ok) {
      const errText = await res.text();
      throw new Error(`HTTP ${res.status}: ${errText}`);
    }

    const result = await res.json();
    console.log('✅ Ответ сервера:', result);
    e.target.reset();
    callback();
  } catch (err) {
    console.error('❌ Ошибка:', err);
    alert(`Ошибка: ${err.message}`);
  }
}

async function fetchChain() {
  try {
    const res = await fetch(`${API}/chain`);
    if (!res.ok) throw new Error(res.statusText);
    const chain = await res.json();
    
    document.getElementById('chain-list').innerHTML = chain.map(r => `
      <div class="block">
        <b>#${r.index}</b> | ${r.type}<br>
        <span class="hash">⏱ ${new Date(r.timestamp).toLocaleString()}</span><br>
        📦 ${JSON.stringify(r.payload)}<br>
        <span class="hash">🔗 ${r.hash.substring(0,16)}...</span>
      </div>
    `).join('');
  } catch (err) {
    console.error('Не удалось загрузить цепь:', err);
  }
}

function showHackResult(res) {
  const box = document.getElementById('hack-result');
  box.classList.remove('hidden');
  box.innerHTML = `
    <b>Статус:</b> ${res.broken_reason === 'none' ? '✅ Целая' : '❌ Цепь сломана'}<br>
    <b>Причина:</b> ${res.broken_reason}<br>
    <b>Следующий блок затронут:</b> ${res.next_broken ? 'Да' : 'Нет'}<br>
    <span class="hash">Новый хеш: ${res.modified_block.hash.substring(0,20)}...</span>
  `;
}