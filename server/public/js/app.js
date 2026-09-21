/**
 * Remote WoL — Frontend Application
 * SPA with auth, device CRUD, WoL triggering, and live status polling
 */

(function () {
    'use strict';

    // ─── State ──────────────────────────────────
    const API = '';
    let token = localStorage.getItem('wol_token') || null;
    let devices = [];
    let statusPollingInterval = null;
    let editingDeviceId = null;
    let deletingDeviceId = null;

    // ─── DOM Elements ───────────────────────────
    const $  = (sel) => document.querySelector(sel);
    const $$ = (sel) => document.querySelectorAll(sel);

    const loginScreen    = $('#login-screen');
    const dashboardScreen = $('#dashboard-screen');
    const loginForm      = $('#login-form');
    const loginError     = $('#login-error');
    const loginBtn       = $('#login-btn');
    const deviceGrid     = $('#device-grid');
    const emptyState     = $('#empty-state');
    const addDeviceBtn   = $('#add-device-btn');
    const refreshBtn     = $('#refresh-btn');
    const logoutBtn      = $('#logout-btn');
    const agentStatus    = $('#agent-status');
    const agentText      = $('.agent-text');

    // Modal elements
    const deviceModal    = $('#device-modal');
    const deviceForm     = $('#device-form');
    const modalTitle     = $('#modal-title');
    const modalClose     = $('#modal-close');
    const modalCancel    = $('#modal-cancel');
    const deleteModal    = $('#delete-modal');
    const deleteConfirm  = $('#delete-confirm');
    const deleteDevName  = $('#delete-device-name');

    // ─── Init ───────────────────────────────────
    function init() {
        if (token) {
            showDashboard();
        } else {
            showLogin();
        }
        bindEvents();
    }

    // ─── Auth ───────────────────────────────────
    function showLogin() {
        loginScreen.classList.add('active');
        dashboardScreen.classList.remove('active');
        stopStatusPolling();
    }

    function showDashboard() {
        loginScreen.classList.remove('active');
        dashboardScreen.classList.add('active');
        loadDevices();
        checkAgentHealth();
        startStatusPolling();
    }

    async function login(username, password) {
        try {
            const res = await api('/api/auth/login', {
                method: 'POST',
                body: JSON.stringify({ username, password }),
                noAuth: true,
            });
            token = res.token;
            localStorage.setItem('wol_token', token);
            showDashboard();
        } catch (err) {
            loginError.textContent = 'Hatalı kullanıcı adı veya şifre';
            shake(loginForm);
        }
    }

    function logout() {
        token = null;
        localStorage.removeItem('wol_token');
        showLogin();
    }

    // ─── API Helper ─────────────────────────────
    async function api(path, opts = {}) {
        const headers = { 'Content-Type': 'application/json' };
        if (!opts.noAuth && token) {
            headers['Authorization'] = `Bearer ${token}`;
        }

        const res = await fetch(`${API}${path}`, {
            method: opts.method || 'GET',
            headers,
            body: opts.body,
        });

        if (res.status === 401) {
            logout();
            throw new Error('Oturum süresi doldu');
        }

        const data = await res.json();
        if (!res.ok) throw new Error(data.error || 'Bir hata oluştu');
        return data;
    }

    // ─── Devices ────────────────────────────────
    async function loadDevices() {
        try {
            devices = await api('/api/devices');
            renderDevices();
            // Load statuses after rendering
            loadAllStatuses();
        } catch (err) {
            toast('Cihazlar yüklenemedi: ' + err.message, 'error');
        }
    }

    function renderDevices() {
        if (!devices || devices.length === 0) {
            deviceGrid.style.display = 'none';
            emptyState.style.display = 'block';
            return;
        }

        deviceGrid.style.display = 'grid';
        emptyState.style.display = 'none';

        deviceGrid.innerHTML = devices.map(dev => {
            const iconName = mapIcon(dev.icon);
            return `
            <div class="device-card" id="device-${dev.id}" data-id="${dev.id}">
                <div class="card-top">
                    <div class="device-icon-wrapper">
                        <span class="material-symbols-rounded">${iconName}</span>
                    </div>
                    <div class="card-actions">
                        <button class="btn btn-icon" onclick="app.editDevice(${dev.id})" title="Düzenle">
                            <span class="material-symbols-rounded">edit</span>
                        </button>
                        <button class="btn btn-icon" onclick="app.confirmDelete(${dev.id})" title="Sil">
                            <span class="material-symbols-rounded">delete</span>
                        </button>
                    </div>
                </div>
                <div class="device-name">${escapeHtml(dev.name)}</div>
                <div class="device-mac">${escapeHtml(dev.mac_address)}</div>
                <div class="device-meta">
                    ${dev.ip_address ? `
                    <div class="meta-item">
                        <span class="material-symbols-rounded">lan</span>
                        ${escapeHtml(dev.ip_address)}
                    </div>` : ''}
                    <div class="meta-item">
                        <span class="material-symbols-rounded">settings_ethernet</span>
                        ${escapeHtml(dev.net_interface)}
                    </div>
                    <div class="meta-item status-placeholder" id="status-${dev.id}">
                        <span class="status-badge checking">
                            <span class="status-dot"></span>
                            Kontrol...
                        </span>
                    </div>
                </div>
                <div class="card-footer">
                    <button class="btn btn-wake" id="wake-btn-${dev.id}" onclick="app.wakeDevice(${dev.id})">
                        <span class="material-symbols-rounded">power_settings_new</span>
                        Uyandır
                    </button>
                    ${dev.ip_address ? `
                    <button class="btn btn-status" onclick="app.checkStatus(${dev.id})" title="Durum Kontrol">
                        <span class="material-symbols-rounded">monitor_heart</span>
                    </button>` : ''}
                </div>
            </div>`;
        }).join('');
    }

    async function loadAllStatuses() {
        try {
            const statuses = await api('/api/devices/status/all');
            statuses.forEach(s => updateStatusBadge(s.device_id, s.online, s.error));
        } catch {
            // Agent likely offline, badges stay as "checking"
        }
    }

    function updateStatusBadge(deviceId, online, error) {
        const el = $(`#status-${deviceId}`);
        const card = $(`#device-${deviceId}`);
        if (!el) return;

        if (error) {
            el.innerHTML = `<span class="status-badge offline"><span class="status-dot"></span>Bilinmiyor</span>`;
            card?.classList.remove('online');
        } else if (online) {
            el.innerHTML = `<span class="status-badge online"><span class="status-dot"></span>Çevrimiçi</span>`;
            card?.classList.add('online');
        } else {
            el.innerHTML = `<span class="status-badge offline"><span class="status-dot"></span>Çevrimdışı</span>`;
            card?.classList.remove('online');
        }
    }

    async function wakeDevice(id) {
        const btn = $(`#wake-btn-${id}`);
        if (!btn || btn.classList.contains('waking')) return;

        const dev = devices.find(d => d.id === id);
        if (!dev) return;

        btn.classList.add('waking');
        btn.innerHTML = '<span class="material-symbols-rounded">hourglass_top</span> Gönderiliyor...';

        try {
            await api(`/api/devices/${id}/wake`, { method: 'POST' });
            toast(`Magic packet gönderildi: ${dev.name}`, 'success');
            btn.innerHTML = '<span class="material-symbols-rounded">hourglass_top</span> Bekleniyor...';

            // Poll for boot if IP is configured
            if (dev.ip_address) {
                pollForBoot(id, dev.name, 20);
            } else {
                setTimeout(() => resetWakeBtn(btn), 3000);
            }
        } catch (err) {
            toast('Hata: ' + err.message, 'error');
            resetWakeBtn(btn);
        }
    }

    async function pollForBoot(id, name, maxAttempts) {
        const btn = $(`#wake-btn-${id}`);
        let attempts = 0;

        const check = async () => {
            if (attempts >= maxAttempts) {
                toast(`⚠️ ${name}: ${maxAttempts * 3} saniye içinde yanıt alınamadı`, 'warning');
                resetWakeBtn(btn);
                return;
            }
            attempts++;

            try {
                const res = await api(`/api/devices/${id}/status`);
                if (res.online) {
                    toast(`✅ ${name} başarıyla açıldı!`, 'success');
                    updateStatusBadge(id, true);
                    resetWakeBtn(btn);
                    return;
                }
            } catch { /* ignore */ }

            setTimeout(check, 3000);
        };

        setTimeout(check, 3000);
    }

    function resetWakeBtn(btn) {
        if (!btn) return;
        btn.classList.remove('waking');
        btn.innerHTML = '<span class="material-symbols-rounded">power_settings_new</span> Uyandır';
    }

    async function checkStatus(id) {
        const dev = devices.find(d => d.id === id);
        if (!dev || !dev.ip_address) return;

        updateStatusBadge(id, false);
        const el = $(`#status-${id}`);
        if (el) el.innerHTML = '<span class="status-badge checking"><span class="status-dot"></span>Kontrol...</span>';

        try {
            const res = await api(`/api/devices/${id}/status`);
            updateStatusBadge(id, res.online);
        } catch (err) {
            updateStatusBadge(id, false, err.message);
        }
    }

    // ─── Device CRUD Modal ──────────────────────
    function openAddModal() {
        editingDeviceId = null;
        modalTitle.textContent = 'Yeni Cihaz Ekle';
        deviceForm.reset();
        $('#device-iface').value = 'br-lan';
        setActiveIcon('desktop_windows');
        deviceModal.style.display = 'flex';
    }

    async function editDevice(id) {
        const dev = devices.find(d => d.id === id);
        if (!dev) return;

        editingDeviceId = id;
        modalTitle.textContent = 'Cihazı Düzenle';
        $('#device-name').value = dev.name;
        $('#device-mac').value = dev.mac_address;
        $('#device-ip').value = dev.ip_address || '';
        $('#device-iface').value = dev.net_interface || 'br-lan';
        setActiveIcon(mapIcon(dev.icon));
        deviceModal.style.display = 'flex';
    }

    function closeModal() {
        deviceModal.style.display = 'none';
        editingDeviceId = null;
    }

    async function saveDevice(e) {
        e.preventDefault();

        const activeIcon = $('.icon-option.active');
        const data = {
            name: $('#device-name').value.trim(),
            mac_address: $('#device-mac').value.trim().toUpperCase(),
            ip_address: $('#device-ip').value.trim(),
            net_interface: $('#device-iface').value.trim() || 'br-lan',
            icon: activeIcon ? activeIcon.dataset.icon : 'desktop_windows',
        };

        try {
            if (editingDeviceId) {
                await api(`/api/devices/${editingDeviceId}`, { method: 'PUT', body: JSON.stringify(data) });
                toast('Cihaz güncellendi', 'success');
            } else {
                await api('/api/devices', { method: 'POST', body: JSON.stringify(data) });
                toast('Cihaz eklendi', 'success');
            }
            closeModal();
            loadDevices();
        } catch (err) {
            toast('Hata: ' + err.message, 'error');
        }
    }

    function confirmDelete(id) {
        const dev = devices.find(d => d.id === id);
        if (!dev) return;
        deletingDeviceId = id;
        deleteDevName.textContent = dev.name;
        deleteModal.style.display = 'flex';
    }

    function closeDeleteModal() {
        deleteModal.style.display = 'none';
        deletingDeviceId = null;
    }

    async function deleteDevice() {
        if (!deletingDeviceId) return;
        try {
            await api(`/api/devices/${deletingDeviceId}`, { method: 'DELETE' });
            toast('Cihaz silindi', 'success');
            closeDeleteModal();
            loadDevices();
        } catch (err) {
            toast('Hata: ' + err.message, 'error');
        }
    }

    // ─── Agent Health ───────────────────────────
    async function checkAgentHealth() {
        try {
            const res = await api('/api/devices/agent/health');
            if (res.connected) {
                agentStatus.className = 'agent-badge connected';
                agentText.textContent = 'Agent bağlı';
            } else {
                agentStatus.className = 'agent-badge disconnected';
                agentText.textContent = 'Bağlantı yok';
            }
        } catch {
            agentStatus.className = 'agent-badge disconnected';
            agentText.textContent = 'Bağlantı yok';
        }
    }

    // ─── Status Polling ─────────────────────────
    function startStatusPolling() {
        stopStatusPolling();
        statusPollingInterval = setInterval(() => {
            loadAllStatuses();
            checkAgentHealth();
        }, 30000);
    }

    function stopStatusPolling() {
        if (statusPollingInterval) {
            clearInterval(statusPollingInterval);
            statusPollingInterval = null;
        }
    }

    // ─── Icon Helpers ───────────────────────────
    function mapIcon(icon) {
        const map = {
            'desktop': 'desktop_windows',
            'desktop_windows': 'desktop_windows',
            'laptop': 'laptop',
            'server': 'dns',
            'dns': 'dns',
            'storage': 'storage',
            'tv': 'smart_display',
            'smart_display': 'smart_display',
            'game': 'videogame_asset',
            'videogame_asset': 'videogame_asset',
        };
        return map[icon] || 'desktop_windows';
    }

    function setActiveIcon(iconName) {
        $$('.icon-option').forEach(btn => {
            btn.classList.toggle('active', btn.dataset.icon === iconName);
        });
    }

    // ─── Toast ──────────────────────────────────
    function toast(message, type = 'info') {
        const container = $('#toast-container');
        const icons = {
            success: 'check_circle',
            error: 'error',
            warning: 'warning',
            info: 'info',
        };

        const el = document.createElement('div');
        el.className = `toast ${type}`;
        el.innerHTML = `
            <span class="material-symbols-rounded">${icons[type] || 'info'}</span>
            <span>${escapeHtml(message)}</span>
        `;
        container.appendChild(el);

        setTimeout(() => {
            el.classList.add('removing');
            el.addEventListener('animationend', () => el.remove());
        }, 4000);
    }

    // ─── Utilities ──────────────────────────────
    function escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str || '';
        return div.innerHTML;
    }

    function shake(el) {
        el.style.animation = 'none';
        el.offsetHeight; // force reflow
        el.style.animation = 'shake 0.4s ease';
    }

    // Add shake keyframes dynamically
    const shakeStyle = document.createElement('style');
    shakeStyle.textContent = `
        @keyframes shake {
            0%, 100% { transform: translateX(0); }
            20% { transform: translateX(-8px); }
            40% { transform: translateX(8px); }
            60% { transform: translateX(-4px); }
            80% { transform: translateX(4px); }
        }
    `;
    document.head.appendChild(shakeStyle);

    // ─── Event Bindings ─────────────────────────
    function bindEvents() {
        loginForm.addEventListener('submit', (e) => {
            e.preventDefault();
            loginError.textContent = '';
            const user = $('#login-username').value.trim();
            const pass = $('#login-password').value;
            login(user, pass);
        });

        logoutBtn.addEventListener('click', logout);
        addDeviceBtn.addEventListener('click', openAddModal);
        refreshBtn.addEventListener('click', () => {
            loadDevices();
            checkAgentHealth();
            toast('Yenilendi', 'info');
        });

        modalClose.addEventListener('click', closeModal);
        modalCancel.addEventListener('click', closeModal);
        deviceForm.addEventListener('submit', saveDevice);

        // Icon picker
        $$('.icon-option').forEach(btn => {
            btn.addEventListener('click', () => {
                setActiveIcon(btn.dataset.icon);
            });
        });

        // Delete modal
        deleteConfirm.addEventListener('click', deleteDevice);
        $$('.delete-modal-close').forEach(btn => {
            btn.addEventListener('click', closeDeleteModal);
        });

        // Close modals on overlay click
        deviceModal.addEventListener('click', (e) => {
            if (e.target === deviceModal) closeModal();
        });
        deleteModal.addEventListener('click', (e) => {
            if (e.target === deleteModal) closeDeleteModal();
        });

        // ESC key to close modals
        document.addEventListener('keydown', (e) => {
            if (e.key === 'Escape') {
                closeModal();
                closeDeleteModal();
            }
        });
    }

    // ─── Expose API for inline handlers ─────────
    window.app = {
        wakeDevice,
        editDevice,
        confirmDelete,
        checkStatus,
    };

    // ─── Boot ───────────────────────────────────
    init();
})();
