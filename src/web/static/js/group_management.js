// Group management functions for RFID Card Management System

function showGroupManagementModal(e) {
    e.preventDefault();

    const groupsTable = `
        <div class="management-actions mb-lg">
            <button id="add-group-btn" class="primary-button">Új csoport</button>
            <div class="search-container">
                <span class="search-icon">🔍</span>
                <input type="text" class="search-input" id="group-search" placeholder="Keresés...">
            </div>
        </div>
        <div class="table-responsive">
            <table id="groups-table">
                <thead>
                    <tr>
                        <th style="width: 30%">Név</th>
                        <th style="width: 40%">Leírás</th>
                        <th style="width: 30%">Műveletek</th>
                    </tr>
                </thead>
                <tbody>
                    ${groups.map(group => `
                        <tr>
                            <td>${group.name}</td>
                            <td>${group.description || 'Nincs leírás'}</td>
                            <td class="actions">
                                <button class="edit-group" data-id="${group.id}">Szerkesztés</button>
                                <button class="delete-group" data-id="${group.id}">Törlés</button>
                                <button class="manage-members" data-id="${group.id}">Tagok</button>
                            </td>
                        </tr>
                    `).join('')}
                </tbody>
            </table>
        </div>
    `;

    showModal('Csoportok kezelése', groupsTable);

    setTimeout(() => {
        document.getElementById('add-group-btn').addEventListener('click', showAddGroupForm);

        document.querySelectorAll('.edit-group').forEach(button => {
            button.addEventListener('click', () => {
                const groupId = button.getAttribute('data-id');
                showEditGroupForm(groupId);
            });
        });

        document.querySelectorAll('.delete-group').forEach(button => {
            button.addEventListener('click', async () => {
                const groupId = button.getAttribute('data-id');
                if (confirm('Biztosan törölni szeretné ezt a csoportot?')) {
                    await deleteGroup(groupId);
                }
            });
        });

        document.querySelectorAll('.manage-members').forEach(button => {
            button.addEventListener('click', () => {
                const groupId = button.getAttribute('data-id');
                showGroupMembersModal(groupId);
            });
        });

        // Keresés beállítása
        const searchInput = document.getElementById('group-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                const searchText = e.target.value.toLowerCase();
                const table = document.getElementById('groups-table');
                if (!table) return;

                const rows = table.querySelectorAll('tbody tr');
                let hasVisibleRows = false;

                rows.forEach(row => {
                    const nameCell = row.querySelector('td:nth-child(1)');
                    const descriptionCell = row.querySelector('td:nth-child(2)');

                    if (!nameCell || !descriptionCell) return;

                    const name = nameCell.textContent.toLowerCase();
                    const description = descriptionCell.textContent.toLowerCase();

                    const isVisible = name.includes(searchText) || description.includes(searchText);
                    row.style.display = isVisible ? '' : 'none';

                    if (isVisible) hasVisibleRows = true;

                    // Ha nincs keresési szöveg, visszaállítjuk az eredeti szöveget
                    if (searchText === '') {
                        nameCell.innerHTML = nameCell.textContent;
                        descriptionCell.innerHTML = descriptionCell.textContent;
                    } else {
                        // Kiemeljük a keresett szöveget
                        nameCell.innerHTML = highlightText(nameCell.textContent, searchText);
                        descriptionCell.innerHTML = highlightText(descriptionCell.textContent, searchText);
                    }
                });

                // Nincs találat üzenet megjelenítése
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

                // Scrollbar ellenőrzése a keresés után
                if (typeof checkScrollbars === 'function') {
                    setTimeout(checkScrollbars, 50);
                }
            });
        }
    }, 0);
}

function showAddGroupForm() {
    const groupForm = `
        <form id="group-form">
            <div class="form-group">
                <label for="name">Név</label>
                <input type="text" id="name" name="name" required placeholder="A csoport neve">
                <div class="form-hint">Adjon meg egy egyedi, könnyen azonosítható nevet</div>
            </div>
            <div class="form-group">
                <label for="description">Leírás</label>
                <input type="text" id="description" name="description" placeholder="A csoport rövid leírása">
                <div class="form-hint">Opcionális mező a csoport céljának, funkciójának leírására</div>
            </div>
            <div class="form-group">
                <label for="parent-group">Szülő csoport (opcionális)</label>
                <select id="parent-group" name="parent-group">
                    <option value="">Nincs szülő csoport</option>
                    ${groups.map(group => `<option value="${group.id}">${group.name}</option>`).join('')}
                </select>
                <div class="form-hint">Válasszon szülő csoportot a hierarchikus struktúrához</div>
            </div>
        </form>
    `;

    const handleConfirm = async () => {
        const name = document.getElementById('name').value;
        const description = document.getElementById('description').value;
        const parentGroupSelect = document.getElementById('parent-group');
        const parentId = parentGroupSelect.value ? Number(parentGroupSelect.value) : null;
        const accessLevel = "restricted"; // Minden csoport csak restricted lehet

        // Részletes hibaellenőrzés
        if (!name) {
            alert('Kérjük, adja meg a csoport nevét!');
            return;
        }

        try {
            const response = await fetch('/api/groups', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                body: JSON.stringify({
                    name,
                    description,
                    parent_id: parentId,
                    access_level: accessLevel
                })
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Nem sikerült a csoport létrehozása');
            }

            alert('Csoport sikeresen létrehozva!');
            hideModal();
            await fetchGroups();
            showGroupManagementModal(new Event('click'));
        } catch (error) {
            alert('Hiba történt: ' + error.message);
        }
    };

    showModal('Új csoport hozzáadása', groupForm, handleConfirm);
}

async function showEditGroupForm(groupId) {
    try {
        const response = await fetch(`/api/groups/${groupId}?include_parent=true`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a csoport adatainak lekérése');
        }

        const group = await response.json();

        const groupForm = `
            <form id="edit-group-form">
                <div class="form-group">
                    <label for="name">Név</label>
                    <input type="text" id="name" name="name" value="${group.name}" required>
                    <div class="form-hint">Adjon meg egy egyedi, könnyen azonosítható nevet</div>
                </div>
                <div class="form-group">
                    <label for="description">Leírás</label>
                    <input type="text" id="description" name="description" value="${group.description || ''}">
                    <div class="form-hint">Opcionális mező a csoport céljának, funkciójának leírására</div>
                </div>
                <div class="form-group">
                    <label for="parent-group">Szülő csoport (opcionális)</label>
                    <select id="parent-group" name="parent-group">
                        <option value="">Nincs szülő csoport</option>
                        ${groups
                            .filter(g => g.id !== group.id) // Kizárjuk az aktuális csoportot
                            .map(g => `<option value="${g.id}" ${group.parent_id && g.id === group.parent_id ? 'selected' : ''}>${g.name}</option>`)
                            .join('')}
                    </select>
                    <div class="form-hint">Válasszon szülő csoportot a hierarchikus struktúrához</div>
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const name = document.getElementById('name').value;
            const description = document.getElementById('description').value;
            const parentGroupSelect = document.getElementById('parent-group');
            const parentId = parentGroupSelect.value ? Number(parentGroupSelect.value) : null;
            const accessLevel = "restricted"; // Minden csoport csak restricted lehet

            // Részletes hibaellenőrzés
            if (!name) {
                alert('Kérjük, adja meg a csoport nevét!');
                return;
            }

            try {
                const response = await fetch(`/api/groups/${groupId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        name,
                        description,
                        parent_id: parentId,
                        access_level: accessLevel
                    })
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a csoport frissítése');
                }

                alert('Csoport sikeresen frissítve!');
                hideModal();
                await fetchGroups();
                showGroupManagementModal(new Event('click'));
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Csoport szerkesztése', groupForm, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function deleteGroup(groupId) {
    try {
        const response = await fetch(`/api/groups/${groupId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a csoport törlése');
        }

        alert('Csoport sikeresen törölve!');
        await fetchGroups();
        showGroupManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function showGroupMembersModal(groupId) {
    try {
        // Lekérjük a csoport adatait
        const response = await fetch(`/api/groups/${groupId}?include_users=true&include_rooms=true`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a csoport adatainak lekérése');
        }

        const group = await response.json();

        // Tabfüles UI a felhasználók és helyiségek között váltáshoz
        const membersContent = `
            <div class="tabs">
                <div class="tab-header">
                    <div class="tab-btn active" data-tab="users">Felhasználók</div>
                    <div class="tab-btn" data-tab="rooms">Helyiségek</div>
                </div>
                <div class="tab-content">
                    <div class="tab-pane active" id="users-tab">
                        <div class="management-actions mb-lg">
                            <button id="add-user-to-group-btn" class="primary-button">Felhasználó hozzáadása</button>
                        </div>
                        <h3>Csoporttagok</h3>
                        <table>
                            <thead>
                                <tr>
                                    <th style="width: 40%">Név</th>
                                    <th style="width: 35%">Felhasználónév</th>
                                    <th style="width: 25%">Műveletek</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${group.users && group.users.length ?
                                    group.users.map(user => `
                                        <tr>
                                            <td>${user.first_name} ${user.last_name}</td>
                                            <td>${user.username}</td>
                                            <td class="actions">
                                                <button class="remove-user-from-group" data-user-id="${user.id}">Eltávolítás</button>
                                            </td>
                                        </tr>
                                    `).join('') :
                                    '<tr><td colspan="3" class="text-center">Nincsenek felhasználók a csoportban</td></tr>'
                                }
                            </tbody>
                        </table>
                    </div>
                    <div class="tab-pane" id="rooms-tab">
                        <div class="management-actions mb-lg">
                            <button id="add-room-to-group-btn" class="primary-button">Helyiség hozzáadása</button>
                        </div>
                        <h3>Csoporthoz rendelt helyiségek</h3>
                        <table>
                            <thead>
                                <tr>
                                    <th style="width: 25%">Név</th>
                                    <th style="width: 20%">Épület</th>
                                    <th style="width: 15%">Teremszám</th>
                                    <th style="width: 20%">Hozzáférési szint</th>
                                    <th style="width: 20%">Műveletek</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${group.rooms && group.rooms.length ?
                                    group.rooms.map(room => `
                                        <tr>
                                            <td>${room.name}</td>
                                            <td>${room.building}</td>
                                            <td>${room.room_number}</td>
                                            <td><span class="badge ${getBadgeClassForAccessLevel(room.access_level)}">${translateAccessLevel(room.access_level)}</span></td>
                                            <td class="actions">
                                                <button class="remove-room-from-group" data-room-id="${room.id}">Eltávolítás</button>
                                            </td>
                                        </tr>
                                    `).join('') :
                                    '<tr><td colspan="5" class="text-center">Nincsenek helyiségek a csoporthoz rendelve</td></tr>'
                                }
                            </tbody>
                        </table>
                    </div>
                </div>
            </div>
        `;

        showModal(`"${group.name}" csoport kezelése`, membersContent);

        // Eseménykezelők beállítása
        setTimeout(() => {
            // Tab funkciók
            document.querySelectorAll('.tab-btn').forEach(tab => {
                tab.addEventListener('click', () => {
                    // Aktív tab frissítése
                    document.querySelectorAll('.tab-btn').forEach(t => t.classList.remove('active'));
                    tab.classList.add('active');

                    // Aktív tartalom frissítése
                    document.querySelectorAll('.tab-pane').forEach(p => p.classList.remove('active'));
                    document.getElementById(`${tab.getAttribute('data-tab')}-tab`).classList.add('active');
                });
            });

            // Felhasználó hozzáadása gomb
            document.getElementById('add-user-to-group-btn').addEventListener('click', () => {
                showAddUserToGroupForm(groupId);
            });

            // Helyiség hozzáadása gomb
            document.getElementById('add-room-to-group-btn').addEventListener('click', () => {
                showAddRoomToGroupForm(groupId);
            });

            // Felhasználó eltávolítása gombok
            document.querySelectorAll('.remove-user-from-group').forEach(button => {
                button.addEventListener('click', async () => {
                    const userId = button.getAttribute('data-user-id');
                    if (confirm('Biztosan el szeretné távolítani ezt a felhasználót a csoportból?')) {
                        await removeUserFromGroup(groupId, userId);
                    }
                });
            });

            // Helyiség eltávolítása gombok
            document.querySelectorAll('.remove-room-from-group').forEach(button => {
                button.addEventListener('click', async () => {
                    const roomId = button.getAttribute('data-room-id');
                    if (confirm('Biztosan el szeretné távolítani ezt a helyiséget a csoportból?')) {
                        await removeRoomFromGroup(groupId, roomId);
                    }
                });
            });
        }, 0);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function showAddUserToGroupForm(groupId) {
    try {
        // Lekérjük a csoport adatait, hogy tudjuk, kik vannak már benne
        const groupResponse = await fetch(`/api/groups/${groupId}?include_users=true`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!groupResponse.ok) {
            throw new Error('Nem sikerült a csoport adatainak lekérése');
        }

        const group = await groupResponse.json();

        // Leszűrjük azokat a felhasználókat, akik még nincsenek a csoportban
        const existingUserIds = group.users ? group.users.map(user => user.id) : [];

        const availableUsers = users.filter(user => !existingUserIds.includes(user.id));

        if (availableUsers.length === 0) {
            alert('Nincs több felhasználó, akit hozzáadhatna a csoporthoz.');
            return;
        }

        const addUserForm = `
            <form id="add-user-to-group-form">
                <div class="form-group">
                    <label for="user-select">Felhasználó</label>
                    <select id="user-select" name="user-select" required>
                        <option value="">Válasszon felhasználót</option>
                        ${availableUsers.map(user => `<option value="${user.id}">${user.first_name} ${user.last_name} (${user.username})</option>`).join('')}
                    </select>
                    <div class="form-hint">Válassza ki a csoporthoz adandó felhasználót</div>
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const userSelect = document.getElementById('user-select');
            const userId = userSelect.value;

            if (!userId) {
                alert('Kérjük, válasszon felhasználót!');
                return;
            }

            try {
                const response = await fetch(`/api/groups/${groupId}/users`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        user_id: Number(userId)
                    })
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a felhasználó hozzáadása a csoporthoz');
                }

                alert('Felhasználó sikeresen hozzáadva a csoporthoz!');
                hideModal();
                await fetchGroups();
                showGroupMembersModal(groupId);
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Felhasználó hozzáadása a csoporthoz', addUserForm, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function showAddRoomToGroupForm(groupId) {
    try {
        // Lekérjük a csoport adatait, hogy tudjuk, mely helyiségek vannak már hozzárendelve
        const groupResponse = await fetch(`/api/groups/${groupId}?include_rooms=true`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!groupResponse.ok) {
            throw new Error('Nem sikerült a csoport adatainak lekérése');
        }

        const group = await groupResponse.json();

        // Leszűrjük azokat a helyiségeket, amelyek még nincsenek a csoporthoz rendelve
        const existingRoomIds = group.rooms ? group.rooms.map(room => room.id) : [];

        const availableRooms = rooms.filter(room => !existingRoomIds.includes(room.id));

        if (availableRooms.length === 0) {
            alert('Nincs több helyiség, amit hozzáadhatna a csoporthoz.');
            return;
        }

        const addRoomForm = `
            <form id="add-room-to-group-form">
                <div class="form-group">
                    <label for="room-select">Helyiség</label>
                    <select id="room-select" name="room-select" required>
                        <option value="">Válasszon helyiséget</option>
                        ${availableRooms.map(room => `<option value="${room.id}">${room.name} (${room.building}, ${room.room_number})</option>`).join('')}
                    </select>
                    <div class="form-hint">Válassza ki a csoporthoz rendelendő helyiséget</div>
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const roomSelect = document.getElementById('room-select');
            const roomId = roomSelect.value;

            if (!roomId) {
                alert('Kérjük, válasszon helyiséget!');
                return;
            }

            try {
                const response = await fetch(`/api/groups/${groupId}/rooms`, {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        room_id: Number(roomId)
                    })
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a helyiség hozzáadása a csoporthoz');
                }

                alert('Helyiség sikeresen hozzáadva a csoporthoz!');
                hideModal();
                await fetchGroups();
                showGroupMembersModal(groupId);
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Helyiség hozzáadása a csoporthoz', addRoomForm, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function removeUserFromGroup(groupId, userId) {
    try {
        const response = await fetch(`/api/groups/${groupId}/users/${userId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a felhasználó eltávolítása a csoportból');
        }

        alert('Felhasználó sikeresen eltávolítva a csoportból!');
        await fetchGroups();
        showGroupMembersModal(groupId);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function removeRoomFromGroup(groupId, roomId) {
    try {
        const response = await fetch(`/api/groups/${groupId}/rooms/${roomId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a helyiség eltávolítása a csoportból');
        }

        alert('Helyiség sikeresen eltávolítva a csoportból!');
        await fetchGroups();
        showGroupMembersModal(groupId);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}