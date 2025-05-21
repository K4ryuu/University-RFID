let authToken = null;
let currentUser = null;
let lastActivity = Date.now();
let inactivityTimer = null;
const SESSION_TIMEOUT = 30 * 60 * 1000; // 30 perc inaktivitás után kijelentkezés

const loginContainer = document.getElementById('login-container');
const dashboardContainer = document.getElementById('dashboard-container');
const loginForm = document.getElementById('login-form');
const loginError = document.getElementById('login-error');
const authNav = document.getElementById('auth-nav');
const totalUsersElement = document.getElementById('total-users');
const totalCardsElement = document.getElementById('total-cards');
const totalRoomsElement = document.getElementById('total-rooms');
const accessTodayElement = document.getElementById('access-today');
const recentLogsTable = document.getElementById('recent-logs-table')?.querySelector('tbody');
const modalContainer = document.getElementById('modal-container');
const modalTitle = document.getElementById('modal-title');
const modalBody = document.getElementById('modal-body');
const modalConfirmBtn = document.querySelector('.modal-confirm');
const modalCancelBtn = document.querySelector('.modal-cancel');
const modalCloseBtn = document.querySelector('.modal-close');

let users = [];
let cards = [];
let rooms = [];
let logs = [];
let groups = [];

document.addEventListener('DOMContentLoaded', init);

function init() {
    authToken = localStorage.getItem('authToken');
    if (authToken) {
        fetchCurrentUser();
        startInactivityTimer();
    }

    loginForm.addEventListener('submit', handleLogin);

    modalCloseBtn.addEventListener('click', hideModal);
    modalCancelBtn.addEventListener('click', hideModal);

    document.getElementById('manage-users-btn')?.addEventListener('click', showUserManagementModal);
    document.getElementById('manage-cards-btn')?.addEventListener('click', showCardManagementModal);
    document.getElementById('manage-groups-btn')?.addEventListener('click', showGroupManagementModal);
    document.getElementById('manage-rooms-btn')?.addEventListener('click', showRoomManagementModal);
    document.getElementById('view-logs-btn')?.addEventListener('click', showLogsModal);
    document.getElementById('simulation-btn')?.addEventListener('click', showSimulationModal);

    const yearElement = document.querySelector('footer p');
    if (yearElement) {
        const currentYear = new Date().getFullYear();
        yearElement.textContent = yearElement.textContent.replace('{{ .year }}', currentYear);
    }

    document.addEventListener('click', updateLastActivity);
    document.addEventListener('keypress', updateLastActivity);
    document.addEventListener('mousemove', throttle(updateLastActivity, 60000)); // Csak 1 percenként frissítünk egérmozgásra
    document.addEventListener('scroll', throttle(updateLastActivity, 60000));    // Csak 1 percenként frissítünk görgetésre

    window.addEventListener('load', checkScrollbars);
    window.addEventListener('resize', checkScrollbars);

    const observer = new MutationObserver((mutations) => {
        let shouldCheckScrollbars = false;

        mutations.forEach(mutation => {
            if (mutation.type === 'childList' ||
                (mutation.type === 'attributes' && mutation.attributeName === 'style')) {
                shouldCheckScrollbars = true;
            }
        });

        if (shouldCheckScrollbars) {
            setTimeout(checkScrollbars, 100);
        }
    });

    observer.observe(document.querySelector('main'), {
        childList: true,
        subtree: true,
        attributes: true,
        attributeFilter: ['style']
    });
}

function updateLastActivity() {
    lastActivity = Date.now();
}

function throttle(func, limit) {
    let inThrottle;
    return function() {
        const args = arguments;
        const context = this;
        if (!inThrottle) {
            func.apply(context, args);
            inThrottle = true;
            setTimeout(() => inThrottle = false, limit);
        }
    };
}

function startInactivityTimer() {
    if (inactivityTimer) {
        clearInterval(inactivityTimer);
    }

    inactivityTimer = setInterval(() => {
        const currentTime = Date.now();
        const timeElapsed = currentTime - lastActivity;

        if (timeElapsed >= SESSION_TIMEOUT) {
            clearInterval(inactivityTimer);
            inactivityTimer = null;
            showInactivityLogoutMessage();
            logout();
        }
    }, 60000);
}

function showInactivityLogoutMessage() {
    const timeoutMinutes = SESSION_TIMEOUT / 60000;
    alert(`Az Ön munkamenete ${timeoutMinutes} perc inaktivitás miatt automatikusan lezárult. Kérjük, jelentkezzen be újra.`);
}

async function handleLogin(e) {
    e.preventDefault();
    loginError.textContent = '';

    const username = document.getElementById('username').value;
    const password = document.getElementById('password').value;

    if (!username) {
        loginError.textContent = 'Kérjük, adja meg a felhasználónevét!';
        return;
    }

    if (!password) {
        loginError.textContent = 'Kérjük, adja meg a jelszavát!';
        return;
    }

    try {
        const response = await fetch('/api/auth/login', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ username, password }),
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Sikertelen bejelentkezés');
        }

        const data = await response.json();
        authToken = data.token;
        localStorage.setItem('authToken', authToken);

        try {
            const userResponse = await fetch('/api/auth/me', {
                headers: {
                    'Authorization': `Bearer ${authToken}`
                }
            });

            if (!userResponse.ok) {
                throw new Error('Nem sikerült a felhasználói adatok lekérése');
            }

            const userData = await userResponse.json();

            if (!userData || !userData.id) {
                throw new Error('Érvénytelen felhasználói adatok a szervertől');
            }

            currentUser = userData;

            updateLastActivity();
            startInactivityTimer();

            showDashboard();
            loadDashboardData();
        } catch (error) {
            console.error('Hiba a felhasználói adatok lekérésekor:', error);
            currentUser = data.user || { first_name: 'Ismeretlen', last_name: 'Felhasználó' };

            updateLastActivity();
            startInactivityTimer();

            showDashboard();
            loadDashboardData();
        }
    } catch (error) {
        loginError.textContent = error.message;
    }
}

async function fetchCurrentUser() {
    try {
        const response = await fetch('/api/auth/me', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            if (response.status === 401) {
                logout();
                return;
            }
            throw new Error('Nem sikerült a felhasználói adatok lekérése');
        }

        const userData = await response.json();

        if (!userData || !userData.id) {
            console.error('Érvénytelen felhasználói adatok a szervertől', userData);
            logout();
            return;
        }

        currentUser = userData;
        showDashboard();
        loadDashboardData();
    } catch (error) {
        console.error('Hiba a felhasználói adatok lekérésekor:', error);
        logout();
    }
}

function logout() {
    authToken = null;
    currentUser = null;
    localStorage.removeItem('authToken');

    if (inactivityTimer) {
        clearInterval(inactivityTimer);
        inactivityTimer = null;
    }

    showLogin();
}

function showLogin() {
    document.querySelector('.login-container-wrapper').style.display = 'block';
    dashboardContainer.style.display = 'none';
    updateAuthNav(false);
}

function showDashboard() {
    document.querySelector('.login-container-wrapper').style.display = 'none';
    dashboardContainer.style.display = 'block';
    updateAuthNav(true);
}

function updateAuthNav(isLoggedIn) {
    if (isLoggedIn && currentUser) {
        const firstName = currentUser.first_name || 'Ismeretlen';
        const lastName = currentUser.last_name || 'Felhasználó';

        authNav.innerHTML = `
            <span>Üdvözöljük, ${firstName} ${lastName}</span>
            <a href="#" id="profile-btn">Profilom</a>
            <a href="#" id="logout-btn">Kijelentkezés</a>
        `;
        document.getElementById('profile-btn').addEventListener('click', (e) => {
            e.preventDefault();
            showProfileModal();
        });
        document.getElementById('logout-btn').addEventListener('click', (e) => {
            e.preventDefault();
            logout();
        });
    } else {
        authNav.innerHTML = `<a href="#">Bejelentkezés</a>`;
    }
}

async function loadDashboardData() {
    if (!authToken || !currentUser) return;

    try {
        await Promise.all([
            fetchDashboardStats(),
            fetchRecentLogs(),
            fetchExpiringCards()
        ]);
    } catch (error) {
        console.error('Hiba az irányítópult adatainak betöltésekor:', error);
    }
}

async function fetchUsers() {
    try {
        const response = await fetch('/api/users', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (response.ok) {
            users = await response.json();
            if (totalUsersElement) {
                totalUsersElement.textContent = users.length || 0;
            }
        }

        return users;
    } catch (error) {
        console.error('Hiba a felhasználók lekérésekor:', error);
        return [];
    }
}

async function fetchCards() {
    try {
        const response = await fetch('/api/cards', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (response.ok) {
            cards = await response.json();
            if (totalCardsElement) {
                totalCardsElement.textContent = cards.length || 0;
            }
        }

        return cards;
    } catch (error) {
        console.error('Hiba a kártyák lekérésekor:', error);
        return [];
    }
}

async function fetchRooms() {
    try {
        const response = await fetch('/api/rooms', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (response.ok) {
            rooms = await response.json();
            if (totalRoomsElement) {
                totalRoomsElement.textContent = rooms.length || 0;
            }
        }

        return rooms;
    } catch (error) {
        console.error('Hiba a helyiségek lekérésekor:', error);
        return [];
    }
}

async function fetchGroups() {
    try {
        const response = await fetch('/api/groups', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (response.ok) {
            groups = await response.json();
        }

        return groups;
    } catch (error) {
        console.error('Hiba a csoportok lekérésekor:', error);
        return [];
    }
}

async function fetchDashboardStats() {
    try {
        await Promise.all([
            fetchUsers(),
            fetchCards(),
            fetchRooms(),
            fetchGroups()
        ]);

        const today = new Date().toISOString().slice(0, 10);
        const logsResponse = await fetch(`/api/logs?start_date=${today}&result=granted`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (logsResponse.ok) {
            const todayLogs = await logsResponse.json();
            accessTodayElement.textContent = todayLogs.length || 0;
        }
    } catch (error) {
        console.error('Hiba a statisztikák lekérésekor:', error);
    }
}

async function fetchRecentLogs() {
    try {
        const response = await fetch('/api/logs?limit=10', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a napló lekérése');
        }

        logs = await response.json();

        if (!Array.isArray(logs)) {
            console.error('Expected logs to be an array, got:', typeof logs);
            logs = Array.isArray(logs) ? logs : [];
        }

        renderRecentLogs(logs);
    } catch (error) {
        console.error('Hiba a naplók lekérésekor:', error);
    }
}

function renderRecentLogs(logs) {
    if (!recentLogsTable) {
        console.error('Recent logs table not found');
        return;
    }

    recentLogsTable.innerHTML = '';

    if (!logs || logs.length === 0) {
        const emptyRow = document.createElement('tr');
        const emptyCell = document.createElement('td');
        emptyCell.setAttribute('colspan', '5');
        emptyCell.className = 'text-center';
        emptyCell.textContent = 'Nincs tevékenység';
        emptyRow.appendChild(emptyCell);
        recentLogsTable.appendChild(emptyRow);
        return;
    }

    logs.forEach((log, index) => {
        try {
            const row = document.createElement('tr');
            if (log.access_result === 'granted') {
                row.classList.add('granted-row');
            } else {
                row.classList.add('denied-row');
            }

            const timeCell = document.createElement('td');
            try {
                const timestamp = new Date(log.timestamp || new Date());
                timeCell.textContent = timestamp.toLocaleString('hu-HU');
            } catch (e) {
                console.error("Error formatting timestamp:", e, log.timestamp);
                timeCell.textContent = 'Ismeretlen idő';
            }
            row.appendChild(timeCell);

            const userCell = document.createElement('td');
            let userName = 'Ismeretlen';

            if (log.card && log.card.user && log.card.user.first_name && log.card.user.last_name) {
                userName = `${log.card.user.first_name} ${log.card.user.last_name}`;
            } else if (log.user && log.user.first_name && log.user.last_name) {
                userName = `${log.user.first_name} ${log.user.last_name}`;
            }

            userCell.textContent = userName;
            row.appendChild(userCell);

            const cardCell = document.createElement('td');
            if (log.card && log.card.card_id) {
                cardCell.textContent = log.card.card_id;
            } else {
                cardCell.textContent = 'Ismeretlen';
            }
            row.appendChild(cardCell);

            const roomCell = document.createElement('td');
            if (log.room && log.room.name) {
                roomCell.textContent = log.room.name;
            } else {
                roomCell.textContent = 'Ismeretlen';
            }
            row.appendChild(roomCell);

            const resultCell = document.createElement('td');
            const badgeSpan = document.createElement('span');
            badgeSpan.className = log.access_result === 'granted' ? 'badge badge-success' : 'badge badge-error';
            badgeSpan.textContent = log.access_result === 'granted' ? 'Engedélyezve' : 'Megtagadva';
            resultCell.appendChild(badgeSpan);

            if (log.access_result === 'denied' && log.denial_reason) {
                const infoIcon = document.createElement('span');
                infoIcon.className = 'info-icon';
                infoIcon.innerHTML = 'i';
                infoIcon.title = getDenialReasonText(log.denial_reason);
                resultCell.appendChild(infoIcon);
            }

            row.appendChild(resultCell);

            recentLogsTable.appendChild(row);

        } catch (error) {
            console.error('Error rendering log entry:', error, log);
        }
    });

    checkScrollbars();
    setTimeout(checkScrollbars, 100);
}

function showModal(title, content, confirmCallback = null) {
    modalTitle.textContent = title;
    modalBody.innerHTML = content;

    if (confirmCallback) {
        modalConfirmBtn.style.display = 'block';
        modalConfirmBtn.onclick = confirmCallback;
    } else {
        modalConfirmBtn.style.display = 'none';
    }

    modalContainer.classList.remove('modal-hidden');
}

function hideModal() {
    modalContainer.classList.add('modal-hidden');
    modalBody.innerHTML = '';
}

function showCardManagementModal(e) {
    e.preventDefault();
    alert('A kártyakezelő funkció jelenleg fejlesztés alatt áll.');
}

function showGroupManagementModal(e) {
    e.preventDefault();
    alert('A csoportkezelő funkció jelenleg fejlesztés alatt áll.');
}


function showUserManagementModal(e) {
    e.preventDefault();

    const usersTable = `
        <div class="management-actions mb-lg">
            <button id="add-user-btn" class="primary-button">Új felhasználó</button>
            <div class="search-container">
                <span class="search-icon">🔍</span>
                <input type="text" class="search-input" id="user-search" placeholder="Keresés...">
            </div>
        </div>
        <div class="table-responsive">
            <table id="users-table">
                <thead>
                    <tr>
                        <th style="width: 20%">Név</th>
                        <th style="width: 15%">Felhasználónév</th>
                        <th style="width: 30%">Email</th>
                        <th style="width: 15%">Adminisztrátor</th>
                        <th style="width: 20%">Műveletek</th>
                    </tr>
                </thead>
                <tbody>
                    ${users.map(user => `
                        <tr>
                            <td>${user.first_name} ${user.last_name}</td>
                            <td>${user.username}</td>
                            <td>${user.email}</td>
                            <td><span class="badge ${user.is_admin ? 'badge-primary' : 'badge-light'}">${user.is_admin ? 'Igen' : 'Nem'}</span></td>
                            <td class="actions">
                                <button class="edit-user" data-id="${user.id}">Szerkesztés</button>
                                <button class="delete-user" data-id="${user.id}">Törlés</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;

    showModal('Felhasználók kezelése', usersTable);

    setTimeout(() => {
        document.getElementById('add-user-btn').addEventListener('click', showAddUserForm);

        document.querySelectorAll('.edit-user').forEach(button => {
            button.addEventListener('click', () => {
                const userId = button.getAttribute('data-id');
                showEditUserForm(userId);
            });
        });

        document.querySelectorAll('.delete-user').forEach(button => {
            button.addEventListener('click', async () => {
                const userId = button.getAttribute('data-id');
                if (confirm('Biztosan törölni szeretné ezt a felhasználót?')) {
                    await deleteUser(userId);
                }
            });
        });

        const searchInput = document.getElementById('user-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                const searchText = e.target.value.toLowerCase();
                const table = document.getElementById('users-table');
                if (!table) return;

                const rows = table.querySelectorAll('tbody tr');
                let hasVisibleRows = false;

                rows.forEach(row => {
                    const nameCell = row.querySelector('td:nth-child(1)');
                    const usernameCell = row.querySelector('td:nth-child(2)');
                    const emailCell = row.querySelector('td:nth-child(3)');

                    if (!nameCell || !usernameCell || !emailCell) return;

                    const name = nameCell.textContent.toLowerCase();
                    const username = usernameCell.textContent.toLowerCase();
                    const email = emailCell.textContent.toLowerCase();

                    const isVisible = name.includes(searchText) || username.includes(searchText) || email.includes(searchText);
                    row.style.display = isVisible ? '' : 'none';

                    if (isVisible) hasVisibleRows = true;

                    if (searchText === '') {
                        nameCell.innerHTML = nameCell.textContent;
                        usernameCell.innerHTML = usernameCell.textContent;
                        emailCell.innerHTML = emailCell.textContent;
                    } else {
                        nameCell.innerHTML = highlightText(nameCell.textContent, searchText);
                        usernameCell.innerHTML = highlightText(usernameCell.textContent, searchText);
                        emailCell.innerHTML = highlightText(emailCell.textContent, searchText);
                    }
                });

                let noResultsMessage = table.parentNode.querySelector('.no-results');
                if (!hasVisibleRows && searchText !== '') {
                    if (!noResultsMessage) {
                        noResultsMessage = document.createElement('div');
                        noResultsMessage.className = 'no-results';
                        noResultsMessage.textContent = 'Nincs találat a keresési feltételeknek megfelelően';
                        table.parentNode.appendChild(noResultsMessage);
                    }
                    noResultsMessage.style.display = 'block';
                } else if (noResultsMessage) {
                    noResultsMessage.style.display = 'none';
                }

                setTimeout(checkScrollbars, 50);
            });
        }
    }, 0);
}

async function showAddUserForm() {
    await fetchGroups();
    await fetchRooms();

    const userForm = `
        <form id="user-form">
            <div class="form-row">
                <div class="form-group">
                    <label for="first-name">Keresztnév</label>
                    <input type="text" id="first-name" name="first-name" required placeholder="Keresztnév">
                </div>
                <div class="form-group">
                    <label for="last-name">Vezetéknév</label>
                    <input type="text" id="last-name" name="last-name" required placeholder="Vezetéknév">
                </div>
            </div>

            <div class="form-group">
                <label for="new-username">Felhasználónév</label>
                <input type="text" id="new-username" name="username" required placeholder="Felhasználónév bejelentkezéshez">
                <div class="form-hint">Egyedi felhasználónév, amelyet bejelentkezéskor használ</div>
            </div>

            <div class="form-group">
                <label for="new-password">Jelszó</label>
                <input type="password" id="new-password" name="password" required placeholder="Jelszó">
                <div class="form-hint">Erős jelszó ajánlott (min. 8 karakter, vegyesen betűk és számok)</div>
            </div>

            <div class="form-group">
                <label for="new-email">Email</label>
                <input type="email" id="new-email" name="email" required placeholder="name@example.com">
            </div>

            <div class="form-row">
                <div class="form-group">
                    <label for="is-admin">Adminisztrátor</label>
                    <input type="checkbox" id="is-admin" name="is-admin">
                    <div class="form-hint">Adminisztrátorok hozzáférhetnek az összes funkcióhoz</div>
                </div>
                <div class="form-group">
                    <label for="is-active">Aktív</label>
                    <input type="checkbox" id="is-active" name="is-active" checked>
                    <div class="form-hint">Aktív felhasználó tud belépni a rendszerbe</div>
                </div>
            </div>

            <div class="form-divider"></div>
            <h3>Csoporttagság</h3>

            <div class="form-group">
                <label>Csoportok</label>
                <div class="checkbox-group" id="group-checkboxes">
                    ${groups.map(group => `
                        <div class="checkbox-item">
                            <input type="checkbox" id="group-${group.id}" name="group-${group.id}" value="${group.id}">
                            <label for="group-${group.id}">${group.name}</label>
                        </div>
                    `).join('')}
                    ${groups.length === 0 ? '<p>Nincsenek elérhető csoportok</p>' : ''}
                </div>
            </div>

            <div class="form-divider"></div>
            <h3>Közvetlen helyiség hozzáférés</h3>

            <div class="form-group">
                <label>Helyiségek</label>
                <div class="checkbox-group" id="room-checkboxes">
                    ${rooms.map(room => `
                        <div class="checkbox-item">
                            <input type="checkbox" id="room-${room.id}" name="room-${room.id}" value="${room.id}">
                            <label for="room-${room.id}">${room.name} (${room.building}, ${room.room_number})</label>
                        </div>
                    `).join('')}
                    ${rooms.length === 0 ? '<p>Nincsenek elérhető helyiségek</p>' : ''}
                </div>
            </div>
        </form>
    `;

    const handleConfirm = async () => {
        const username = document.getElementById('new-username').value;
        const password = document.getElementById('new-password').value;
        const firstName = document.getElementById('first-name').value;
        const lastName = document.getElementById('last-name').value;
        const email = document.getElementById('new-email').value;
        const isAdmin = document.getElementById('is-admin').checked;
        const isActive = document.getElementById('is-active').checked;

        const selectedGroups = [];
        document.querySelectorAll('#group-checkboxes input[type="checkbox"]:checked').forEach(checkbox => {
            selectedGroups.push(parseInt(checkbox.value));
        });

        const selectedRooms = [];
        document.querySelectorAll('#room-checkboxes input[type="checkbox"]:checked').forEach(checkbox => {
            selectedRooms.push(parseInt(checkbox.value));
        });

        if (!firstName) {
            alert('Kérjük, adja meg a keresztnevet!');
            return;
        }

        if (!lastName) {
            alert('Kérjük, adja meg a vezetéknevet!');
            return;
        }

        if (!username) {
            alert('Kérjük, adja meg a felhasználónevet!');
            return;
        }

        if (!password) {
            alert('Kérjük, adja meg a jelszót!');
            return;
        }

        if (password.length < 8) {
            alert('A jelszónak legalább 8 karakter hosszúnak kell lennie!');
            return;
        }

        if (!/[A-Z]/.test(password)) {
            alert('A jelszónak tartalmaznia kell legalább egy nagybetűt!');
            return;
        }

        if (!/[a-z]/.test(password)) {
            alert('A jelszónak tartalmaznia kell legalább egy kisbetűt!');
            return;
        }

        if (!/[0-9]/.test(password)) {
            alert('A jelszónak tartalmaznia kell legalább egy számot!');
            return;
        }

        if (!/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)) {
            alert('A jelszónak tartalmaznia kell legalább egy speciális karaktert (pl. !@#$%^&*)!');
            return;
        }

        if (!email) {
            alert('Kérjük, adja meg az email címet!');
            return;
        }

        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(email)) {
            alert('Kérjük, adjon meg egy érvényes email címet!');
            return;
        }

        try {
            const userResponse = await fetch('/api/users', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                body: JSON.stringify({
                    username,
                    password,
                    first_name: firstName,
                    last_name: lastName,
                    email,
                    is_admin: isAdmin,
                    active: isActive
                })
            });

            if (!userResponse.ok) {
                const errorData = await userResponse.json();
                throw new Error(errorData.error || 'Nem sikerült a felhasználó létrehozása');
            }

            const newUser = await userResponse.json();
            const userId = newUser.id;

            const groupPromises = selectedGroups.map(groupId =>
                fetch(`/api/groups/${groupId}/users`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        user_id: userId
                    })
                })
            );

            const roomPromises = selectedRooms.map(roomId =>
                fetch('/api/permissions', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        user_id: userId,
                        room_id: roomId,
                        granted_by: currentUser.id,
                        valid_from: new Date()
                    })
                })
            );

            await Promise.all([...groupPromises, ...roomPromises]);

            alert('Felhasználó sikeresen létrehozva!');
            hideModal();

            await fetchUsers();
            showUserManagementModal(new Event('click'));
        } catch (error) {
            alert('Hiba történt: ' + error.message);
        }
    };

    showModal('Új felhasználó hozzáadása', userForm, handleConfirm);
}

async function showEditUserForm(userId) {
    try {
        const userResponse = await fetch(`/api/users/${userId}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!userResponse.ok) {
            throw new Error('Nem sikerült a felhasználói adatok lekérése');
        }

        const user = await userResponse.json();

        await fetchGroups();
        await fetchRooms();

        const userGroupsResponse = await fetch(`/api/users/${userId}?include_groups=true`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        let userGroups = [];
        if (userGroupsResponse.ok) {
            const userData = await userGroupsResponse.json();
            userGroups = userData.groups || [];
        }

        const userPermissionsResponse = await fetch(`/api/permissions?user_id=${userId}&type=user`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        let userPermissions = [];
        if (userPermissionsResponse.ok) {
            userPermissions = await userPermissionsResponse.json();
        }

        const userRoomIds = userPermissions.map(perm => perm.room_id);

        const userForm = `
            <form id="edit-user-form">
                <div class="form-group">
                    <label for="edit-username">Felhasználónév</label>
                    <input type="text" id="edit-username" name="username" value="${user.username}" required>
                </div>
                <div class="form-group">
                    <label for="edit-password">Új jelszó (üresen hagyva nem változik)</label>
                    <input type="password" id="edit-password" name="password">
                </div>
                <div class="form-group">
                    <label for="edit-first-name">Keresztnév</label>
                    <input type="text" id="edit-first-name" name="first-name" value="${user.first_name}" required>
                </div>
                <div class="form-group">
                    <label for="edit-last-name">Vezetéknév</label>
                    <input type="text" id="edit-last-name" name="last-name" value="${user.last_name}" required>
                </div>
                <div class="form-group">
                    <label for="edit-email">Email</label>
                    <input type="email" id="edit-email" name="email" value="${user.email}" required>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="edit-is-admin">Adminisztrátor</label>
                        <input type="checkbox" id="edit-is-admin" name="is-admin" ${user.is_admin ? 'checked' : ''}>
                    </div>
                    <div class="form-group">
                        <label for="edit-active">Aktív</label>
                        <input type="checkbox" id="edit-active" name="active" ${user.active ? 'checked' : ''}>
                    </div>
                </div>

                <div class="form-divider"></div>
                <h3>Csoporttagság</h3>

                <div class="form-group">
                    <label>Csoportok</label>
                    <div class="checkbox-group" id="group-checkboxes">
                        ${groups.map(group => {
                            const isInGroup = userGroups.some(ug => ug.id === group.id);
                            return `
                                <div class="checkbox-item">
                                    <input type="checkbox" id="group-${group.id}" name="group-${group.id}" value="${group.id}" ${isInGroup ? 'checked' : ''}>
                                    <label for="group-${group.id}">${group.name}</label>
                                </div>
                            `;
                        }).join('')}
                        ${groups.length === 0 ? '<p>Nincsenek elérhető csoportok</p>' : ''}
                    </div>
                </div>

                <div class="form-divider"></div>
                <h3>Közvetlen helyiség hozzáférés</h3>

                <div class="form-group">
                    <label>Helyiségek</label>
                    <div class="checkbox-group" id="room-checkboxes">
                        ${rooms.map(room => {
                            const hasDirectAccess = userRoomIds.includes(room.id);
                            return `
                                <div class="checkbox-item">
                                    <input type="checkbox" id="room-${room.id}" name="room-${room.id}" value="${room.id}" ${hasDirectAccess ? 'checked' : ''}>
                                    <label for="room-${room.id}">${room.name} (${room.building}, ${room.room_number})</label>
                                </div>
                            `;
                        }).join('')}
                        ${rooms.length === 0 ? '<p>Nincsenek elérhető helyiségek</p>' : ''}
                    </div>
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const username = document.getElementById('edit-username').value;
            const password = document.getElementById('edit-password').value;
            const firstName = document.getElementById('edit-first-name').value;
            const lastName = document.getElementById('edit-last-name').value;
            const email = document.getElementById('edit-email').value;
            const isAdmin = document.getElementById('edit-is-admin').checked;
            const active = document.getElementById('edit-active').checked;

            const selectedGroups = [];
            document.querySelectorAll('#group-checkboxes input[type="checkbox"]:checked').forEach(checkbox => {
                selectedGroups.push(parseInt(checkbox.value));
            });

            const selectedRooms = [];
            document.querySelectorAll('#room-checkboxes input[type="checkbox"]:checked').forEach(checkbox => {
                selectedRooms.push(parseInt(checkbox.value));
            });

            if (!username) {
                alert('Kérjük, adja meg a felhasználónevet!');
                return;
            }

            if (!firstName) {
                alert('Kérjük, adja meg a keresztnevet!');
                return;
            }

            if (!lastName) {
                alert('Kérjük, adja meg a vezetéknevet!');
                return;
            }

            if (!email) {
                alert('Kérjük, adja meg az email címet!');
                return;
            }

            const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
            if (!emailRegex.test(email)) {
                alert('Kérjük, adjon meg egy érvényes email címet!');
                return;
            }

            const userData = {
                username,
                first_name: firstName,
                last_name: lastName,
                email,
                is_admin: isAdmin,
                active
            };

            if (password) {
                userData.password = password;
            }

            try {
                const response = await fetch(`/api/users/${userId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify(userData)
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a felhasználó frissítése');
                }

                const groupsToRemove = userGroups.filter(g => !selectedGroups.includes(g.id));
                const groupRemovePromises = groupsToRemove.map(group =>
                    fetch(`/api/groups/${group.id}/users/${userId}`, {
                        method: 'DELETE',
                        headers: {
                            'Authorization': `Bearer ${authToken}`
                        }
                    })
                );

                const existingGroupIds = userGroups.map(g => g.id);
                const groupsToAdd = selectedGroups.filter(groupId => !existingGroupIds.includes(groupId));
                const groupAddPromises = groupsToAdd.map(groupId =>
                    fetch(`/api/groups/${groupId}/users`, {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': `Bearer ${authToken}`
                        },
                        body: JSON.stringify({
                            user_id: userId
                        })
                    })
                );

                const roomsToRemove = userRoomIds.filter(roomId => !selectedRooms.includes(roomId));
                const userPermissionsToRevoke = userPermissions.filter(perm => roomsToRemove.includes(perm.room_id));
                const permissionRevokePromises = userPermissionsToRevoke.map(perm =>
                    fetch(`/api/permissions/${perm.id}/revoke`, {
                        method: 'POST',
                        headers: {
                            'Authorization': `Bearer ${authToken}`
                        }
                    })
                );

                const roomsToAdd = selectedRooms.filter(roomId => !userRoomIds.includes(roomId));
                const permissionAddPromises = roomsToAdd.map(roomId =>
                    fetch('/api/permissions', {
                        method: 'POST',
                        headers: {
                            'Content-Type': 'application/json',
                            'Authorization': `Bearer ${authToken}`
                        },
                        body: JSON.stringify({
                            user_id: userId,
                            room_id: roomId,
                            granted_by: currentUser.id,
                            valid_from: new Date()
                        })
                    })
                );

                await Promise.all([
                    ...groupRemovePromises,
                    ...groupAddPromises,
                    ...permissionRevokePromises,
                    ...permissionAddPromises
                ]);

                alert('Felhasználó sikeresen frissítve!');
                hideModal();

                await fetchUsers();
                showUserManagementModal(new Event('click'));
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Felhasználó szerkesztése', userForm, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function deleteUser(userId) {
    try {
        const response = await fetch(`/api/users/${userId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a felhasználó törlése');
        }

        alert('Felhasználó sikeresen törölve!');

        await fetchUsers();
        showUserManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

function showRoomManagementModal(e) {
    e.preventDefault();

    const roomsTable = `
        <div class="management-actions mb-lg">
            <button id="add-room-btn" class="primary-button">Új helyiség</button>
            <div class="search-container">
                <span class="search-icon">🔍</span>
                <input type="text" class="search-input" id="room-search" placeholder="Keresés...">
            </div>
        </div>
        <div class="table-responsive">
            <table id="rooms-table">
                <thead>
                    <tr>
                        <th style="width: 18%">Név</th>
                        <th style="width: 16%">Épület</th>
                        <th style="width: 14%">Teremszám</th>
                        <th style="width: 20%">Hozzáférési szint</th>
                        <th style="width: 32%">Műveletek</th>
                    </tr>
                </thead>
                <tbody>
                    ${rooms.map(room => `
                        <tr>
                            <td>${room.name}</td>
                            <td>${room.building}</td>
                            <td>${room.room_number}</td>
                            <td><span class="badge ${getBadgeClassForAccessLevel(room.access_level)}">${translateAccessLevel(room.access_level)}</span></td>
                            <td class="actions">
                                <button class="edit-room" data-id="${room.id}">Szerkesztés</button>
                                <button class="delete-room" data-id="${room.id}">Törlés</button>
                                <button class="view-permissions" data-id="${room.id}">Jogosultságok</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;

    showModal('Helyiségek kezelése', roomsTable);

    setTimeout(() => {
        document.getElementById('add-room-btn').addEventListener('click', showAddRoomForm);

        document.querySelectorAll('.edit-room').forEach(button => {
            button.addEventListener('click', () => {
                const roomId = button.getAttribute('data-id');
                showEditRoomForm(roomId);
            });
        });

        document.querySelectorAll('.delete-room').forEach(button => {
            button.addEventListener('click', async () => {
                const roomId = button.getAttribute('data-id');
                if (confirm('Biztosan törölni szeretné ezt a helyiséget?')) {
                    await deleteRoom(roomId);
                }
            });
        });

        document.querySelectorAll('.view-permissions').forEach(button => {
            button.addEventListener('click', () => {
                const roomId = button.getAttribute('data-id');
                showRoomPermissions(roomId);
            });
        });

        const searchInput = document.getElementById('room-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                const searchText = e.target.value.toLowerCase();
                const table = document.getElementById('rooms-table');
                if (!table) return;

                const rows = table.querySelectorAll('tbody tr');
                let hasVisibleRows = false;

                rows.forEach(row => {
                    const nameCell = row.querySelector('td:nth-child(1)');
                    const buildingCell = row.querySelector('td:nth-child(2)');
                    const roomNumberCell = row.querySelector('td:nth-child(3)');

                    if (!nameCell || !buildingCell || !roomNumberCell) return;

                    const name = nameCell.textContent.toLowerCase();
                    const building = buildingCell.textContent.toLowerCase();
                    const roomNumber = roomNumberCell.textContent.toLowerCase();

                    const isVisible = name.includes(searchText) || building.includes(searchText) || roomNumber.includes(searchText);
                    row.style.display = isVisible ? '' : 'none';

                    if (isVisible) hasVisibleRows = true;

                    if (searchText === '') {
                        nameCell.innerHTML = nameCell.textContent;
                        buildingCell.innerHTML = buildingCell.textContent;
                        roomNumberCell.innerHTML = roomNumberCell.textContent;
                    } else {
                        nameCell.innerHTML = highlightText(nameCell.textContent, searchText);
                        buildingCell.innerHTML = highlightText(buildingCell.textContent, searchText);
                        roomNumberCell.innerHTML = highlightText(roomNumberCell.textContent, searchText);
                    }
                });

                let noResultsMessage = table.parentNode.querySelector('.no-results');
                if (!hasVisibleRows && searchText !== '') {
                    if (!noResultsMessage) {
                        noResultsMessage = document.createElement('div');
                        noResultsMessage.className = 'no-results';
                        noResultsMessage.textContent = 'Nincs találat a keresési feltételeknek megfelelően';
                        table.parentNode.appendChild(noResultsMessage);
                    }
                    noResultsMessage.style.display = 'block';
                } else if (noResultsMessage) {
                    noResultsMessage.style.display = 'none';
                }

                setTimeout(checkScrollbars, 50);
            });
        }
    }, 0);
}

function showAddRoomForm() {
    const roomForm = `
        <form id="room-form">
            <div class="form-group">
                <label for="name">Név</label>
                <input type="text" id="name" name="name" required>
            </div>
            <div class="form-group">
                <label for="description">Leírás</label>
                <textarea id="description" name="description"></textarea>
            </div>
            <div class="form-group">
                <label for="building">Épület</label>
                <input type="text" id="building" name="building" required>
            </div>
            <div class="form-group">
                <label for="room-number">Teremszám</label>
                <input type="text" id="room-number" name="room-number" required>
            </div>
            <div class="form-group">
                <label for="access-level">Hozzáférési szint</label>
                <select id="access-level" name="access-level" required>
                    <option value="public">Nyilvános</option>
                    <option value="restricted" selected>Engedéllyel</option>
                </select>
            </div>
            <div class="form-group">
                <label for="capacity">Kapacitás</label>
                <input type="number" id="capacity" name="capacity" min="0">
            </div>
            <div class="form-group">
                <label for="operating-hours">Nyitvatartási idő (pl. 08:00-18:00)</label>
                <input type="text" id="operating-hours" name="operating-hours">
            </div>
            <div class="form-group">
                <label for="operating-days">Nyitvatartási napok (0=vasárnap, 1=hétfő, ...)</label>
                <input type="text" id="operating-days" name="operating-days" placeholder="pl. 1,2,3,4,5">
            </div>
        </form>
    `;

    const handleConfirm = async () => {
        const name = document.getElementById('name').value;
        const description = document.getElementById('description').value;
        const building = document.getElementById('building').value;
        const roomNumber = document.getElementById('room-number').value;
        const accessLevel = document.getElementById('access-level').value;
        const capacityEl = document.getElementById('capacity');
        const capacity = capacityEl.value ? parseInt(capacityEl.value) : 0;
        const operatingHours = document.getElementById('operating-hours').value;
        const operatingDays = document.getElementById('operating-days').value;

        if (!name) {
            alert('Kérjük, adja meg a helyiség nevét!');
            return;
        }

        if (!building) {
            alert('Kérjük, adja meg az épület nevét!');
            return;
        }

        if (!roomNumber) {
            alert('Kérjük, adja meg a teremszámot!');
            return;
        }

        try {
            const response = await fetch('/api/rooms', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                body: JSON.stringify({
                    name,
                    description,
                    building,
                    room_number: roomNumber,
                    access_level: accessLevel,
                    capacity,
                    operating_hours: operatingHours,
                    operating_days: operatingDays
                })
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Nem sikerült a helyiség létrehozása');
            }

            alert('Helyiség sikeresen létrehozva!');
            hideModal();
            await fetchRooms();
            showRoomManagementModal(new Event('click'));
        } catch (error) {
            alert('Hiba történt: ' + error.message);
        }
    };

    showModal('Új helyiség hozzáadása', roomForm, handleConfirm);
}

async function showEditRoomForm(roomId) {
    try {
        const response = await fetch(`/api/rooms/${roomId}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a helyiség adatainak lekérése');
        }

        const room = await response.json();

        const roomForm = `
            <form id="edit-room-form">
                <div class="form-group">
                    <label for="name">Név</label>
                    <input type="text" id="name" name="name" value="${room.name}" required>
                </div>
                <div class="form-group">
                    <label for="description">Leírás</label>
                    <textarea id="description" name="description">${room.description || ''}</textarea>
                </div>
                <div class="form-group">
                    <label for="building">Épület</label>
                    <input type="text" id="building" name="building" value="${room.building}" required>
                </div>
                <div class="form-group">
                    <label for="room-number">Teremszám</label>
                    <input type="text" id="room-number" name="room-number" value="${room.room_number}" required>
                </div>
                <div class="form-group">
                    <label for="access-level">Hozzáférési szint</label>
                    <select id="access-level" name="access-level" required>
                        <option value="public" ${room.access_level === 'public' ? 'selected' : ''}>Nyilvános</option>
                        <option value="restricted" ${room.access_level === 'restricted' ? 'selected' : ''}>Engedéllyel</option>
                    </select>
                </div>
                <div class="form-group">
                    <label for="capacity">Kapacitás</label>
                    <input type="number" id="capacity" name="capacity" min="0" value="${room.capacity || 0}">
                </div>
                <div class="form-group">
                    <label for="operating-hours">Nyitvatartási idő (pl. 08:00-18:00)</label>
                    <input type="text" id="operating-hours" name="operating-hours" value="${room.operating_hours || ''}">
                </div>
                <div class="form-group">
                    <label for="operating-days">Nyitvatartási napok (0=vasárnap, 1=hétfő, ...)</label>
                    <input type="text" id="operating-days" name="operating-days" value="${room.operating_days || ''}" placeholder="pl. 1,2,3,4,5">
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const name = document.getElementById('name').value;
            const description = document.getElementById('description').value;
            const building = document.getElementById('building').value;
            const roomNumber = document.getElementById('room-number').value;
            const accessLevel = document.getElementById('access-level').value;
            const capacityEl = document.getElementById('capacity');
            const capacity = capacityEl.value ? parseInt(capacityEl.value) : 0;
            const operatingHours = document.getElementById('operating-hours').value;
            const operatingDays = document.getElementById('operating-days').value;

            if (!name) {
                alert('Kérjük, adja meg a helyiség nevét!');
                return;
            }

            if (!building) {
                alert('Kérjük, adja meg az épület nevét!');
                return;
            }

            if (!roomNumber) {
                alert('Kérjük, adja meg a teremszámot!');
                return;
            }

            try {
                const response = await fetch(`/api/rooms/${roomId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        name,
                        description,
                        building,
                        room_number: roomNumber,
                        access_level: accessLevel,
                        capacity,
                        operating_hours: operatingHours,
                        operating_days: operatingDays
                    })
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a helyiség frissítése');
                }

                alert('Helyiség sikeresen frissítve!');
                hideModal();
                await fetchRooms();
                showRoomManagementModal(new Event('click'));
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Helyiség szerkesztése', roomForm, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function deleteRoom(roomId) {
    try {
        const response = await fetch(`/api/rooms/${roomId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a helyiség törlése');
        }

        alert('Helyiség sikeresen törölve!');
        await fetchRooms();
        showRoomManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function showRoomPermissions(roomId) {
    try {
        const response = await fetch(`/api/rooms/${roomId}/permissions`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a jogosultságok lekérése');
        }

        const permissions = await response.json();
        const room = rooms.find(r => r.id.toString() === roomId);

        const permissionsTable = `
            <div class="management-actions">
                <button id="add-permission-btn" class="primary-button" data-room-id="${roomId}">Új jogosultság</button>
            </div>
            <h3>${room ? room.name : 'Helyiség'} jogosultságai</h3>
            <table>
                <thead>
                    <tr>
                        <th style="width: 20%">Felhasználó</th>
                        <th style="width: 15%">Kártya</th>
                        <th style="width: 15%">Érvényes kezdete</th>
                        <th style="width: 15%">Érvényes vége</th>
                        <th style="width: 10%">Aktív</th>
                        <th style="width: 25%">Műveletek</th>
                    </tr>
                </thead>
                <tbody>
                    ${permissions.length === 0 ? '<tr><td colspan="6">Nincsenek jogosultságok</td></tr>' : ''}
                    ${permissions.map(perm => `
                        <tr>
                            <td>${perm.card && perm.card.user ? `${perm.card.user.first_name} ${perm.card.user.last_name}` : 'Ismeretlen'}</td>
                            <td>${perm.card ? perm.card.card_id : 'Ismeretlen'}</td>
                            <td>${new Date(perm.valid_from).toLocaleDateString('hu-HU')}</td>
                            <td>${perm.valid_until ? new Date(perm.valid_until).toLocaleDateString('hu-HU') : 'Nincs lejárat'}</td>
                            <td>${perm.active ? 'Igen' : 'Nem'}</td>
                            <td class="actions">
                                <button class="edit-permission" data-id="${perm.id}">Szerkesztés</button>
                                <button class="revoke-permission" data-id="${perm.id}">Visszavonás</button>
                                <button class="delete-permission" data-id="${perm.id}">Törlés</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        `;

        showModal('Helyiség jogosultságok', permissionsTable);

        setTimeout(() => {
            document.getElementById('add-permission-btn').addEventListener('click', () => {
                showAddPermissionForm(roomId);
            });

            document.querySelectorAll('.edit-permission').forEach(button => {
                button.addEventListener('click', () => {
                    const permId = button.getAttribute('data-id');
                    showEditPermissionForm(permId);
                });
            });

            document.querySelectorAll('.revoke-permission').forEach(button => {
                button.addEventListener('click', async () => {
                    const permId = button.getAttribute('data-id');
                    if (confirm('Biztosan vissza szeretné vonni ezt a jogosultságot?')) {
                        await revokePermission(permId);
                    }
                });
            });

            document.querySelectorAll('.delete-permission').forEach(button => {
                button.addEventListener('click', async () => {
                    const permId = button.getAttribute('data-id');
                    if (confirm('Biztosan törölni szeretné ezt a jogosultságot?')) {
                        await deletePermission(permId);
                    }
                });
            });
        }, 0);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

function showLogsModal(e) {
    e.preventDefault();

    const today = new Date().toISOString().slice(0, 10);
    const weekAgo = new Date();
    weekAgo.setDate(weekAgo.getDate() - 7);
    const weekAgoStr = weekAgo.toISOString().slice(0, 10);

    const logsContent = `
        <div class="card mb-lg">
            <div class="card-header">
                <h3 class="card-title">Szűrési feltételek</h3>
            </div>
            <div class="logs-filters">
                <div class="form-row">
                    <div class="form-group">
                        <label for="start-date">Kezdődátum</label>
                        <input type="date" id="start-date" name="start-date" value="${weekAgoStr}">
                    </div>
                    <div class="form-group">
                        <label for="end-date">Végdátum</label>
                        <input type="date" id="end-date" name="end-date" value="${today}">
                    </div>
                </div>
                <div class="form-row">
                    <div class="form-group">
                        <label for="room-filter">Helyiség</label>
                        <select id="room-filter" name="room-filter">
                            <option value="">Összes helyiség</option>
                            ${rooms.map(room => `<option value="${room.id}">${room.name}</option>`).join('')}
                        </select>
                    </div>
                    <div class="form-group">
                        <label for="result-filter">Eredmény</label>
                        <select id="result-filter" name="result-filter">
                            <option value="">Összes</option>
                            <option value="granted">Engedélyezve</option>
                            <option value="denied">Megtagadva</option>
                        </select>
                    </div>
                </div>
                <div class="text-right">
                    <button id="apply-filters-btn" class="primary-button">Szűrők alkalmazása</button>
                </div>
            </div>
        </div>
        <div id="logs-table-container">
            <div class="management-actions mb-lg">
                <h3 class="mb-0">Naplóbejegyzések</h3>
                <div class="search-container">
                    <span class="search-icon">🔍</span>
                    <input type="text" class="search-input" id="log-search" placeholder="Keresés a találatokban...">
                </div>
            </div>
            <div class="table-responsive">
                <table id="logs-table">
                    <thead>
                        <tr>
                            <th style="width: 20%">Időpont</th>
                            <th style="width: 20%">Felhasználó</th>
                            <th style="width: 20%">Kártya azonosító</th>
                            <th style="width: 20%">Helyiség</th>
                            <th style="width: 20%">Eredmény</th>
                        </tr>
                    </thead>
                    <tbody id="logs-table-body">
                        <tr>
                            <td colspan="5" class="text-center">Válasszon szűrőket, majd kattintson a "Szűrők alkalmazása" gombra</td>
                        </tr>
                    </tbody>
                </table>
            </div>
        </div>
    `;

    showModal('Belépési napló', logsContent);

    setTimeout(() => {
        document.getElementById('apply-filters-btn').addEventListener('click', fetchFilteredLogs);

        const searchInput = document.getElementById('log-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                const searchText = e.target.value.toLowerCase();
                const tableBody = document.getElementById('logs-table-body');
                if (!tableBody) return;

                const rows = tableBody.querySelectorAll('tr');
                let hasVisibleRows = false;

                rows.forEach(row => {
                    if (row.querySelector('td[colspan]')) {
                        return;
                    }

                    const cells = row.querySelectorAll('td');
                    let rowVisible = false;

                    cells.forEach(cell => {
                        const cellText = cell.textContent.toLowerCase();
                        if (cellText.includes(searchText)) {
                            rowVisible = true;
                        }

                        if (searchText === '') {
                            cell.innerHTML = cell.textContent;
                        } else {
                            if (cell.querySelector('.badge')) {
                                const badge = cell.querySelector('.badge').outerHTML;
                                const textContent = cell.textContent;
                                cell.innerHTML = badge + ' ' + highlightText(textContent.replace(cell.querySelector('.badge').textContent, ''), searchText);
                            } else {
                                cell.innerHTML = highlightText(cell.textContent, searchText);
                            }
                        }
                    });

                    row.style.display = rowVisible || searchText === '' ? '' : 'none';
                    if (rowVisible || searchText === '') hasVisibleRows = true;
                });

                const table = document.getElementById('logs-table');
                let noResultsMessage = table.parentNode.querySelector('.no-results');

                if (!hasVisibleRows && searchText !== '') {
                    if (!noResultsMessage) {
                        noResultsMessage = document.createElement('div');
                        noResultsMessage.className = 'no-results';
                        noResultsMessage.textContent = 'Nincs találat a keresési feltételeknek megfelelően';
                        table.parentNode.appendChild(noResultsMessage);
                    }
                    noResultsMessage.style.display = 'block';
                } else if (noResultsMessage) {
                    noResultsMessage.style.display = 'none';
                }

                setTimeout(checkScrollbars, 50);
            });
        }
    }, 0);
}

async function fetchFilteredLogs() {
    const startDate = document.getElementById('start-date').value;
    const endDate = document.getElementById('end-date').value;
    const roomId = document.getElementById('room-filter').value;
    const result = document.getElementById('result-filter').value;

    let url = '/api/logs?';

    if (startDate) {
        url += `start_date=${startDate}&`;
    }

    if (endDate) {
        url += `end_date=${endDate}&`;
    }

    if (roomId) {
        url += `room_id=${roomId}&`;
    }

    if (result) {
        url += `result=${result}&`;
    }

    try {
        const response = await fetch(url, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a naplók lekérése');
        }

        const filteredLogs = await response.json();
        const logsTableBody = document.getElementById('logs-table-body');

        logsTableBody.innerHTML = '';

        if (!filteredLogs || filteredLogs.length === 0) {
            const row = document.createElement('tr');
            row.innerHTML = '<td colspan="5" class="text-center">Nincsenek naplóbejegyzések a megadott szűrőkkel</td>';
            logsTableBody.appendChild(row);
            return;
        }

        filteredLogs.forEach(log => {
            const row = document.createElement('tr');

            const timestamp = new Date(log.timestamp);
            const formattedTime = timestamp.toLocaleString('hu-HU');

            const userName = log.card && log.card.user ?
                `${log.card.user.first_name} ${log.card.user.last_name}` : 'Ismeretlen';

            const roomName = log.room ? log.room.name : 'Ismeretlen';

            const badgeClass = log.access_result === 'granted' ? 'badge-success' : 'badge-error';
            const resultText = log.access_result === 'granted' ? 'Engedélyezve' : 'Megtagadva';

            row.innerHTML = `
                <td>${formattedTime}</td>
                <td>${userName}</td>
                <td>${log.card ? log.card.card_id : 'Ismeretlen'}</td>
                <td>${roomName}</td>
                <td><span class="badge ${badgeClass}">${resultText}</span></td>
            `;

            logsTableBody.appendChild(row);
        });

        const searchInput = document.getElementById('log-search');
        if (searchInput) {
            searchInput.value = '';
        }

        setTimeout(checkScrollbars, 50);

    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

function translateRole(role) {
    const roles = {
        'admin': 'Adminisztrátor',
        'faculty': 'Oktató',
        'student': 'Hallgató',
        'guest': 'Vendég'
    };

    return roles[role] || role;
}

function translateAccessLevel(level) {
    const levels = {
        'public': 'Nyilvános',
        'restricted': 'Engedéllyel',
    };

    return levels[level] || level;
}

async function fetchExpiringCards() {
    try {
        const response = await fetch('/api/cards/expiring', {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a lejáró kártyák lekérése');
        }

        const expiringCards = await response.json();
        renderExpiringCards(expiringCards);
    } catch (error) {
        console.error('Hiba a lejáró kártyák lekérésekor:', error);
    }
}

function renderExpiringCards(cards) {
    const expiringCardsContainer = document.getElementById('expiring-cards-container');

    if (!expiringCardsContainer) {
        console.error('Lejáró kártyák konténer nem található');
        return;
    }

    expiringCardsContainer.innerHTML = '';

    if (!cards || cards.length === 0) {
        expiringCardsContainer.innerHTML = '<p class="info-message">Nincsenek hamarosan lejáró kártyák</p>';
        return;
    }

    cards.sort((a, b) => new Date(a.expiry_date) - new Date(b.expiry_date));

    const tableHtml = `
        <div class="table-responsive">
            <table id="expiring-cards-table">
                <thead>
                    <tr>
                        <th>Felhasználó</th>
                        <th>Kártya azonosító</th>
                        <th>Lejárati dátum</th>
                        <th>Hátralevő idő</th>
                        <th>Művelet</th>
                    </tr>
                </thead>
                <tbody>
                    ${cards.map(card => {
                        const expiryDate = new Date(card.expiry_date);
                        const today = new Date();
                        const daysRemaining = Math.ceil((expiryDate - today) / (1000 * 60 * 60 * 24));
                        const userName = card.user ? `${card.user.first_name} ${card.user.last_name}` : 'Ismeretlen';

                        let rowClass = '';
                        if (daysRemaining <= 7) {
                            rowClass = 'urgent-row';
                        } else if (daysRemaining <= 30) {
                            rowClass = 'warning-row';
                        }

                        return `
                            <tr class="${rowClass}">
                                <td>${userName}</td>
                                <td>${card.card_id}</td>
                                <td>${expiryDate.toLocaleDateString('hu-HU')}</td>
                                <td><span class="badge ${daysRemaining <= 7 ? 'badge-error' : 'badge-warning'}">${daysRemaining} nap</span></td>
                                <td><button class="extend-card" data-id="${card.id}">Hosszabbítás</button></td>
                            </tr>
                        `;
                    }).join('')}
                </tbody>
            </table>
        </div>
    `;

    expiringCardsContainer.innerHTML = tableHtml;

    setTimeout(() => {
        document.querySelectorAll('.extend-card').forEach(button => {
            button.addEventListener('click', () => {
                const cardId = button.getAttribute('data-id');
                showExtendCardForm(cardId);
            });
        });

        checkScrollbars();
    }, 50);
}

async function showExtendCardForm(cardId) {
    try {
        const response = await fetch(`/api/cards/${cardId}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a kártya adatok lekérése');
        }

        const card = await response.json();
        const currentExpiry = card.expiry_date ? new Date(card.expiry_date) : new Date();

        const defaultNewExpiry = new Date();
        defaultNewExpiry.setFullYear(defaultNewExpiry.getFullYear() + 1);

        const formatDate = (date) => {
            const year = date.getFullYear();
            const month = String(date.getMonth() + 1).padStart(2, '0');
            const day = String(date.getDate()).padStart(2, '0');
            return `${year}-${month}-${day}`;
        };

        const userName = card.user ? `${card.user.first_name} ${card.user.last_name}` : 'Ismeretlen';

        const formContent = `
            <form id="extend-card-form">
                <div class="card extend-card-summary mb-lg">
                    <div class="card-header">
                        <h3 class="card-title">Kártya adatai</h3>
                    </div>
                    <div class="extend-card-info">
                        <div class="extend-card-user">
                            <strong>${userName}</strong>
                            <span class="badge badge-primary">${translateCardStatus(card.status || 'active')}</span>
                        </div>
                        <div class="extend-card-details">
                            <div class="detail-row">
                                <span class="detail-label">Kártya ID:</span>
                                <span class="detail-value">${card.card_id}</span>
                            </div>
                            <div class="detail-row">
                                <span class="detail-label">Jelenlegi lejárat:</span>
                                <span class="detail-value">${currentExpiry.toLocaleDateString('hu-HU')}</span>
                            </div>
                            <div class="detail-row">
                                <span class="detail-label">Kiadás dátuma:</span>
                                <span class="detail-value">${new Date(card.issue_date).toLocaleDateString('hu-HU')}</span>
                            </div>
                        </div>
                    </div>
                </div>

                <div class="form-divider"></div>

                <div class="form-group">
                    <label for="new-expiry-date">Új lejárati dátum</label>
                    <input type="date" id="new-expiry-date" name="new-expiry-date" value="${formatDate(defaultNewExpiry)}" required>
                    <div class="form-hint">A kártya új lejárati dátuma. Az alapértelmezett érték 1 év a mai naptól.</div>
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const newExpiryStr = document.getElementById('new-expiry-date').value;
            if (!newExpiryStr) {
                alert('Kérjük, adja meg az új lejárati dátumot!');
                return;
            }

            const newExpiry = new Date(newExpiryStr);
            const today = new Date();

            if (newExpiry <= today) {
                alert('Az új lejárati dátumnak a mai napnál későbbinek kell lennie!');
                return;
            }

            try {
                const response = await fetch(`/api/cards/${cardId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        expiry_date: newExpiry.toISOString()
                    })
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a kártya hosszabbítása');
                }

                alert('Kártya sikeresen hosszabbítva!');
                hideModal();

                await Promise.all([
                    fetchCards(),
                    fetchExpiringCards()
                ]);
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Kártya hosszabbítása', formContent, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

function getBadgeClassForAccessLevel(level) {
    switch(level) {
        case 'public': return 'badge-success';
        case 'restricted': return 'badge-warning';
        case 'admin': return 'badge-error';
        default: return 'badge-primary';
    }
}

function getDenialReasonText(reason) {
    const reasons = {
        'no_permission': 'Nincs jogosultság a helyiséghez',
        'card_inactive': 'Inaktív kártya',
        'card_expired': 'Lejárt kártya',
        'outside_hours': 'Nyitvatartási időn kívül',
        'room_closed': 'Helyiség zárva',
        'card_blocked': 'Zárolt kártya',
        'card_revoked': 'Visszavont kártya',
        'permission_error': 'Jogosultság hiba'
    };

    return reasons[reason] || reason || 'Ismeretlen ok';
}

function highlightText(text, searchQuery) {
    if (!searchQuery) return text;

    const lowerText = text.toLowerCase();
    const lowerQuery = searchQuery.toLowerCase();

    if (!lowerText.includes(lowerQuery)) return text;

    const startIndex = lowerText.indexOf(lowerQuery);
    const endIndex = startIndex + lowerQuery.length;

    return (
        text.substring(0, startIndex) +
        '<span class="highlight">' + text.substring(startIndex, endIndex) + '</span>' +
        text.substring(endIndex)
    );
}

function checkScrollbars() {
    document.querySelectorAll('.table-responsive').forEach(container => {
        const table = container.querySelector('table');
        if (!table) return;

        if (table.offsetHeight > container.offsetHeight) {
            container.classList.add('needs-scroll');
        } else {
            container.classList.remove('needs-scroll');
        }

        if (table.offsetWidth > container.offsetWidth) {
            container.classList.add('needs-scroll-x');
        } else {
            container.classList.remove('needs-scroll-x');
        }
    });
}

function checkPasswordStrength(password) {
    let score = 0;

    if (password.length === 0) {
        return {
            score: 0,
            text: "Nincs megadva"
        };
    }

    if (password.length >= 8) score += 1;
    if (password.length >= 10) score += 1;
    if (password.length >= 12) score += 1;

    if (/[0-9]/.test(password)) score += 1;

    if (/[a-z]/.test(password)) score += 1;

    if (/[A-Z]/.test(password)) score += 1;

    if (/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)) score += 1;

    const uniqueChars = new Set(password.split('')).size;
    if (uniqueChars >= 5) score += 1;
    if (uniqueChars >= 8) score += 1;

    let text, strengthClass;

    if (score < 3) {
        text = "Nagyon gyenge";
        strengthClass = "very-weak";
    } else if (score < 5) {
        text = "Gyenge";
        strengthClass = "weak";
    } else if (score < 7) {
        text = "Közepes";
        strengthClass = "medium";
    } else if (score < 9) {
        text = "Erős";
        strengthClass = "strong";
    } else {
        text = "Nagyon erős";
        strengthClass = "very-strong";
    }

    return {
        score: score,
        text: text,
        strengthClass: strengthClass
    };
}

function showProfileModal() {
    const profileForm = `
        <form id="profile-form">
            <div class="form-row">
                <div class="form-group">
                    <label for="profile-first-name">Keresztnév</label>
                    <input type="text" id="profile-first-name" name="first-name" value="${currentUser.first_name}" required>
                </div>
                <div class="form-group">
                    <label for="profile-last-name">Vezetéknév</label>
                    <input type="text" id="profile-last-name" name="last-name" value="${currentUser.last_name}" required>
                </div>
            </div>
            <div class="form-group">
                <label for="profile-email">Email</label>
                <input type="email" id="profile-email" name="email" value="${currentUser.email}" required>
            </div>
            <div class="form-divider"></div>
            <h3>Jelszó módosítása</h3>
            <div class="form-group">
                <label for="profile-current-password">Jelenlegi jelszó</label>
                <input type="password" id="profile-current-password" name="current-password" placeholder="Adja meg jelenlegi jelszavát">
                <div class="form-hint">Jelszóváltoztatáshoz adja meg jelenlegi jelszavát</div>
            </div>
            <div class="form-group">
                <label for="profile-password">Új jelszó</label>
                <input type="password" id="profile-password" name="password" placeholder="Új jelszó">
                <div class="password-strength-meter">
                    <div class="password-strength-bar" id="password-strength-bar"></div>
                </div>
                <div class="password-strength-text" id="password-strength-text">Jelszóerősség: Nincs megadva</div>
            </div>
            <div class="form-group">
                <label for="profile-password-confirm">Új jelszó megerősítése</label>
                <input type="password" id="profile-password-confirm" name="password-confirm" placeholder="Jelszó megerősítése">
            </div>
        </form>
    `;

    const handleSaveProfile = async () => {
        const firstName = document.getElementById('profile-first-name').value;
        const lastName = document.getElementById('profile-last-name').value;
        const email = document.getElementById('profile-email').value;
        const password = document.getElementById('profile-password').value;
        const passwordConfirm = document.getElementById('profile-password-confirm').value;

        if (!firstName) {
            alert('Kérjük, adja meg a keresztnevét!');
            return;
        }

        if (!lastName) {
            alert('Kérjük, adja meg a vezetéknevét!');
            return;
        }

        if (!email) {
            alert('Kérjük, adja meg az email címét!');
            return;
        }

        const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
        if (!emailRegex.test(email)) {
            alert('Kérjük, adjon meg egy érvényes email címet!');
            return;
        }

        const currentPassword = document.getElementById('profile-current-password').value;

        if (password || currentPassword) {
            if (!currentPassword) {
                alert('Jelszó módosításához meg kell adnia a jelenlegi jelszavát!');
                return;
            }

            if (!password) {
                alert('Kérjük, adja meg az új jelszót!');
                return;
            }

            if (password.length < 8) {
                alert('A jelszónak legalább 8 karakter hosszúnak kell lennie!');
                return;
            }

            if (!/[A-Z]/.test(password)) {
                alert('A jelszónak tartalmaznia kell legalább egy nagybetűt!');
                return;
            }

            if (!/[a-z]/.test(password)) {
                alert('A jelszónak tartalmaznia kell legalább egy kisbetűt!');
                return;
            }

            if (!/[0-9]/.test(password)) {
                alert('A jelszónak tartalmaznia kell legalább egy számot!');
                return;
            }

            if (!/[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]/.test(password)) {
                alert('A jelszónak tartalmaznia kell legalább egy speciális karaktert (pl. !@#$%^&*)!');
                return;
            }

            if (password !== passwordConfirm) {
                alert('A megadott új jelszavak nem egyeznek!');
                return;
            }
        }

        const userData = {
            first_name: firstName,
            last_name: lastName,
            email: email
        };

        let passwordChanged = false;

        if (password && currentPassword) {
            try {
                const passwordResponse = await fetch('/api/auth/change-password', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        old_password: currentPassword,
                        new_password: password
                    })
                });

                if (!passwordResponse.ok) {
                    if (passwordResponse.status === 401) {
                        alert('A megadott jelenlegi jelszó helytelen!');
                        return;
                    } else {
                        const errorData = await passwordResponse.json();
                        throw new Error(errorData.error || 'Jelszó módosítása sikertelen');
                    }
                }

                passwordChanged = true;
            } catch (error) {
                alert('Hiba a jelszó módosítása során: ' + error.message);
                return;
            }
        }

        try {
            const response = await fetch(`/api/users/${currentUser.id}`, {
                method: 'PUT',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                body: JSON.stringify(userData)
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Profil frissítése sikertelen');
            }

            const updatedUser = await response.json();
            currentUser = updatedUser;

            updateAuthNav(true);

            if (passwordChanged) {
                alert('Profil sikeresen frissítve, és a jelszó megváltoztatva!');
            } else {
                alert('Profil sikeresen frissítve!');
            }
            hideModal();
        } catch (error) {
            alert('Hiba történt: ' + error.message);
        }
    };

    showModal('Profilom szerkesztése', profileForm, handleSaveProfile);

    setTimeout(() => {
        const passwordInput = document.getElementById('profile-password');
        const strengthBar = document.getElementById('password-strength-bar');
        const strengthText = document.getElementById('password-strength-text');

        passwordInput.addEventListener('input', function() {
            const password = this.value;
            const strength = checkPasswordStrength(password);

            strengthBar.classList.remove('very-weak', 'weak', 'medium', 'strong', 'very-strong');

            if (password.length > 0) {
                strengthBar.classList.add(strength.strengthClass);
            }

            strengthText.textContent = `Jelszóerősség: ${strength.text}`;
        });
    }, 0);
}

function showSimulationModal(e) {
    e.preventDefault();

    const simulationContent = `
        <div class="simulation-container">
            <div class="sim-description mb-lg">
                <p>Ez a felület lehetővé teszi a kártyaolvasók és az RFID kártyák szimulációját. Válasszon ki egy kártyát és egy helyiséget, majd indítsa el a szimulációt.</p>
            </div>

            <form id="simulation-form">
                <div class="form-row">
                    <div class="form-group">
                        <label for="sim-card">RFID Kártya</label>
                        <select id="sim-card" name="sim-card" required>
                            <option value="">Válasszon kártyát</option>
                            ${cards.map(card => {
                                const userName = card.user ? `${card.user.first_name} ${card.user.last_name}` : 'Ismeretlen';
                                return `<option value="${card.id}">${card.card_id} - ${userName}</option>`;
                            }).join('')}
                        </select>
                        <div class="form-hint">A szimulációban használni kívánt kártya</div>
                    </div>
                </div>

                <div class="form-row">
                    <div class="form-group">
                        <label for="sim-room">Helyiség</label>
                        <select id="sim-room" name="sim-room" required>
                            <option value="">Válasszon helyiséget</option>
                            ${rooms.map(room => `<option value="${room.id}">${room.name} (${room.building}, ${room.room_number})</option>`).join('')}
                        </select>
                        <div class="form-hint">Az a helyiség, ahol a kártyaolvasót szimulálni szeretné</div>
                    </div>
                </div>
            </form>

            <div class="sim-actions mt-lg">
                <button id="run-simulation-btn" class="primary-button">Belépés szimulálása</button>
            </div>

            <div id="sim-result" class="sim-result mt-xl" style="display: none;">
                <div class="card">
                    <div class="card-header">
                        <h3 class="card-title">Belépési kísérlet eredménye</h3>
                    </div>
                    <div id="sim-result-content" class="card-content mt-lg">
                        <!-- Az eredmények itt jelennek meg -->
                    </div>
                </div>
            </div>
        </div>
    `;

    showModal('Belépési szimuláció', simulationContent);

    setTimeout(() => {
        document.getElementById('run-simulation-btn').addEventListener('click', runAccessSimulation);
    }, 0);
}

async function runAccessSimulation() {
    const cardId = document.getElementById('sim-card').value;
    const roomId = document.getElementById('sim-room').value;

    if (!cardId || !roomId) {
        alert('Kérjük, válasszon ki egy kártyát és egy helyiséget!');
        return;
    }

    try {
        const selectedCard = cards.find(card => card.id.toString() === cardId);
        const selectedRoom = rooms.find(room => room.id.toString() === roomId);

        if (!selectedCard || !selectedRoom) {
            throw new Error('Érvénytelen kártya vagy helyiség kiválasztva');
        }

        const response = await fetch('/api/simulate/access', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Authorization': `Bearer ${authToken}`
            },
            body: JSON.stringify({
                card_id: selectedCard.id,
                room_id: selectedRoom.id
            })
        });

        const result = await response.json();

        const resultContainer = document.getElementById('sim-result');
        const resultContent = document.getElementById('sim-result-content');

        resultContainer.style.display = 'block';

        const userName = selectedCard.user ?
            `${selectedCard.user.first_name} ${selectedCard.user.last_name}` : 'Ismeretlen felhasználó';

        const timestamp = new Date().toLocaleString('hu-HU');
        const accessGranted = result.access_granted;
        const resultClass = accessGranted ? 'success-result' : 'error-result';
        const resultText = accessGranted ? 'Engedélyezve' : 'Megtagadva';
        const badgeClass = accessGranted ? 'badge-success' : 'badge-error';

        resultContent.innerHTML = `
            <div class="sim-result-card ${resultClass}">
                <div class="sim-result-header">
                    <h3>Belépési kísérlet:
                        <span class="badge ${badgeClass}">${resultText}</span>
                        ${!accessGranted && result.reason_code ? `<span class="info-icon" title="${result.reason_text}">i</span>` : ''}
                    </h3>
                    <p class="timestamp">${timestamp}</p>
                </div>
                <div class="sim-result-details">
                    <div class="detail-row">
                        <span class="detail-label">Felhasználó:</span>
                        <span class="detail-value">${userName}</span>
                    </div>
                    <div class="detail-row">
                        <span class="detail-label">Kártya azonosító:</span>
                        <span class="detail-value">${selectedCard.card_id}</span>
                    </div>
                    <div class="detail-row">
                        <span class="detail-label">Helyiség:</span>
                        <span class="detail-value">${selectedRoom.name} (${selectedRoom.building}, ${selectedRoom.room_number})</span>
                    </div>
                    ${!accessGranted ? `
                    <div class="detail-row">
                        <span class="detail-label">Elutasítás oka:</span>
                        <span class="detail-value">${result.reason || 'Nincs megfelelő jogosultság'}</span>
                    </div>
                    ` : ''}
                </div>
            </div>
        `;

        const resultCard = resultContent.querySelector('.sim-result-card');
        resultCard.classList.add('animate-in');

        await fetchRecentLogs();

    } catch (error) {
        console.error('Szimuláció hiba:', error);
        alert('Hiba történt a szimuláció során: ' + error.message);
    }
}