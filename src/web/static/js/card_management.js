// Card management functions for RFID Card Management System

function showCardManagementModal(e) {
    if (e) e.preventDefault();

    // Tabs hozzáadása a különböző nézetekhez
    const cardsTable = `
        <div class="tabs">
            <div class="tab-header">
                <div class="tab-btn active" data-tab="all-cards">Összes kártya</div>
                <div class="tab-btn" data-tab="expiring-cards">Lejáró kártyák</div>
            </div>
            <div class="tab-content">
                <div class="tab-pane active" id="all-cards">
                    <div class="management-actions mb-lg">
                        <button id="add-card-btn" class="primary-button">Új kártya regisztrálása</button>
                        <div class="search-container">
                            <span class="search-icon">🔍</span>
                            <input type="text" class="search-input" id="card-search" placeholder="Keresés...">
                        </div>
                    </div>
                    <div class="table-responsive">
                        <table id="cards-table">
                            <thead>
                                <tr>
                                    <th style="width: 15%">Kártya ID</th>
                                    <th style="width: 25%">Felhasználó</th>
                                    <th style="width: 15%">Státusz</th>
                                    <th style="width: 15%">Lejárati dátum</th>
                                    <th style="width: 30%">Műveletek</th>
                                </tr>
                            </thead>
                            <tbody>
                                ${cards.map(card => {
                                    const user = card.user ? `${card.user.first_name} ${card.user.last_name}` : 'Nincs hozzárendelve';
                                    const statusBadgeClass = getCardStatusBadge(card.status);
                                    const statusText = translateCardStatus(card.status);
                                    const expiryDate = card.expiry_date ? new Date(card.expiry_date).toLocaleDateString('hu-HU') : 'Nincs lejárat';

                                    return `
                                        <tr>
                                            <td>${card.card_id}</td>
                                            <td>${user}</td>
                                            <td><span class="badge ${statusBadgeClass}">${statusText}</span></td>
                                            <td>${expiryDate}</td>
                                            <td class="actions">
                                                <button class="edit-card" data-id="${card.id}">Szerkesztés</button>
                                                <button class="delete-card" data-id="${card.id}">Törlés</button>
                                                ${card.status === 'active' ? `<button class="block-card" data-id="${card.id}">Blokkolás</button>` : ''}
                                                ${card.status === 'blocked' ? `<button class="unblock-card" data-id="${card.id}">Feloldás</button>` : ''}
                                                ${card.status !== 'revoked' ? `<button class="revoke-card" data-id="${card.id}">Visszavonás</button>` : ''}
                                            </td>
                                        </tr>
                                    `;
                                }).join('')}
                            </tbody>
                        </table>
                    </div>
                </div>
                <div class="tab-pane" id="expiring-cards">
                    <div class="management-actions mb-lg">
                        <button id="refresh-expiring-btn" class="primary-button">Frissítés</button>
                        <div class="search-container">
                            <span class="search-icon">🔍</span>
                            <input type="text" class="search-input" id="expiring-card-search" placeholder="Keresés...">
                        </div>
                    </div>
                    <div id="tab-expiring-container">
                        <p class="loading-message">Lejáró kártyák betöltése...</p>
                    </div>
                </div>
            </div>
        </div>
    `;

    showModal('Kártyák kezelése', cardsTable);

    setTimeout(() => {
        // Ellenőrizzük a scrollbarokat a modal megjelenítése után
        if (typeof checkScrollbars === 'function') {
            checkScrollbars();
        }
        
        // Tab kezelés
        document.querySelectorAll('.tab-btn').forEach(tab => {
            tab.addEventListener('click', () => {
                const tabId = tab.getAttribute('data-tab');

                // Tab gomb aktiválása
                document.querySelectorAll('.tab-btn').forEach(t => t.classList.remove('active'));
                tab.classList.add('active');

                // Tab tartalom megjelenítése
                document.querySelectorAll('.tab-pane').forEach(pane => pane.classList.remove('active'));
                document.getElementById(tabId).classList.add('active');

                // Ha a lejáró kártyák tabra váltunk, betöltjük a lejáró kártyákat
                if (tabId === 'expiring-cards') {
                    loadExpiringCardsTab();
                } else {
                    // Tab váltás után mindig ellenőrizzük a scrollbart
                    if (typeof checkScrollbars === 'function') {
                        setTimeout(checkScrollbars, 50);
                    }
                }
            });
        });

        // "Összes kártya" tab eseménykezelői
        document.getElementById('add-card-btn').addEventListener('click', showCardRegistrationModal);

        document.querySelectorAll('.edit-card').forEach(button => {
            button.addEventListener('click', () => {
                const cardId = button.getAttribute('data-id');
                showEditCardForm(cardId);
            });
        });

        document.querySelectorAll('.delete-card').forEach(button => {
            button.addEventListener('click', async () => {
                const cardId = button.getAttribute('data-id');
                if (confirm('Biztosan törölni szeretné ezt a kártyát? Ez az összes kapcsolódó jogosultságot is törölni fogja.')) {
                    await deleteCard(cardId);
                }
            });
        });

        document.querySelectorAll('.block-card').forEach(button => {
            button.addEventListener('click', async () => {
                const cardId = button.getAttribute('data-id');
                if (confirm('Biztosan blokkolni szeretné ezt a kártyát? A kártya ideiglenes letiltásra kerül.')) {
                    await blockCard(cardId);
                }
            });
        });

        document.querySelectorAll('.unblock-card').forEach(button => {
            button.addEventListener('click', async () => {
                const cardId = button.getAttribute('data-id');
                if (confirm('Biztosan fel szeretné oldani ezt a kártyát? A kártya ismét használható lesz.')) {
                    await unblockCard(cardId);
                }
            });
        });

        document.querySelectorAll('.revoke-card').forEach(button => {
            button.addEventListener('click', async () => {
                const cardId = button.getAttribute('data-id');
                if (confirm('Biztosan vissza szeretné vonni ezt a kártyát? Ez a művelet nem visszafordítható.')) {
                    await revokeCard(cardId);
                }
            });
        });

        // "Lejáró kártyák" tab eseménykezelői
        document.getElementById('refresh-expiring-btn')?.addEventListener('click', () => {
            loadExpiringCardsTab();
        });
        
        // Kártyakeresés beállítása - "Összes kártya" tab
        const cardSearchInput = document.getElementById('card-search');
        if (cardSearchInput) {
            cardSearchInput.addEventListener('input', (e) => {
                const searchText = e.target.value.toLowerCase();
                const table = document.getElementById('cards-table');
                if (!table) return;
                
                const rows = table.querySelectorAll('tbody tr');
                let hasVisibleRows = false;
                
                rows.forEach(row => {
                    const cardIdCell = row.querySelector('td:nth-child(1)');
                    const userCell = row.querySelector('td:nth-child(2)');
                    const statusCell = row.querySelector('td:nth-child(3)');
                    const expiryCell = row.querySelector('td:nth-child(4)');
                    
                    if (!cardIdCell || !userCell || !statusCell || !expiryCell) return;
                    
                    const cardId = cardIdCell.textContent.toLowerCase();
                    const user = userCell.textContent.toLowerCase();
                    const status = statusCell.textContent.toLowerCase();
                    const expiry = expiryCell.textContent.toLowerCase();
                    
                    const isVisible = cardId.includes(searchText) || 
                                     user.includes(searchText) || 
                                     status.includes(searchText) || 
                                     expiry.includes(searchText);
                    
                    row.style.display = isVisible ? '' : 'none';
                    
                    if (isVisible) hasVisibleRows = true;
                    
                    // Ha nincs keresési szöveg, visszaállítjuk az eredeti szöveget
                    if (searchText === '') {
                        cardIdCell.innerHTML = cardIdCell.textContent;
                        userCell.innerHTML = userCell.textContent;
                        // A státusz cellában megőrizzük a badge-et
                        const badge = statusCell.querySelector('.badge');
                        if (badge) {
                            statusCell.innerHTML = '';
                            statusCell.appendChild(badge.cloneNode(true));
                        } else {
                            statusCell.innerHTML = statusCell.textContent;
                        }
                        expiryCell.innerHTML = expiryCell.textContent;
                    } else {
                        // Kiemeljük a keresett szöveget
                        cardIdCell.innerHTML = highlightText(cardIdCell.textContent, searchText);
                        userCell.innerHTML = highlightText(userCell.textContent, searchText);
                        
                        // A státusz cellában megőrizzük a badge-et
                        const badge = statusCell.querySelector('.badge');
                        if (badge) {
                            const badgeText = badge.textContent;
                            const badgeHtml = badge.outerHTML.replace(badgeText, highlightText(badgeText, searchText));
                            statusCell.innerHTML = badgeHtml;
                        } else {
                            statusCell.innerHTML = highlightText(statusCell.textContent, searchText);
                        }
                        
                        expiryCell.innerHTML = highlightText(expiryCell.textContent, searchText);
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
        
        // Kártyakeresés beállítása a lejáró kártyák fülhöz
        const expiringCardSearchInput = document.getElementById('expiring-card-search');
        if (expiringCardSearchInput) {
            expiringCardSearchInput.addEventListener('input', (e) => {
                const searchText = e.target.value.toLowerCase();
                
                // A keresés csak akkor működik, ha már betöltődtek a lejáró kártyák
                const table = document.querySelector('#tab-expiring-container table');
                if (!table) return;
                
                const rows = table.querySelectorAll('tbody tr');
                let hasVisibleRows = false;
                
                rows.forEach(row => {
                    const userCell = row.querySelector('td:nth-child(1)');
                    const cardIdCell = row.querySelector('td:nth-child(2)');
                    const expiryCell = row.querySelector('td:nth-child(3)');
                    const daysCell = row.querySelector('td:nth-child(4)');
                    
                    if (!userCell || !cardIdCell || !expiryCell || !daysCell) return;
                    
                    const user = userCell.textContent.toLowerCase();
                    const cardId = cardIdCell.textContent.toLowerCase();
                    const expiry = expiryCell.textContent.toLowerCase();
                    const days = daysCell.textContent.toLowerCase();
                    
                    const isVisible = user.includes(searchText) || 
                                     cardId.includes(searchText) || 
                                     expiry.includes(searchText) ||
                                     days.includes(searchText);
                    
                    row.style.display = isVisible ? '' : 'none';
                    
                    if (isVisible) hasVisibleRows = true;
                    
                    // Ha nincs keresési szöveg, visszaállítjuk az eredeti szöveget
                    if (searchText === '') {
                        userCell.innerHTML = userCell.textContent;
                        cardIdCell.innerHTML = cardIdCell.textContent;
                        expiryCell.innerHTML = expiryCell.textContent;
                        
                        // A napok cellában megőrizzük a badge-et
                        const badge = daysCell.querySelector('.badge');
                        if (badge) {
                            daysCell.innerHTML = '';
                            daysCell.appendChild(badge.cloneNode(true));
                        } else {
                            daysCell.innerHTML = daysCell.textContent;
                        }
                    } else {
                        // Kiemeljük a keresett szöveget
                        userCell.innerHTML = highlightText(userCell.textContent, searchText);
                        cardIdCell.innerHTML = highlightText(cardIdCell.textContent, searchText);
                        expiryCell.innerHTML = highlightText(expiryCell.textContent, searchText);
                        
                        // A napok cellában megőrizzük a badge-et
                        const badge = daysCell.querySelector('.badge');
                        if (badge) {
                            const badgeText = badge.textContent;
                            const badgeHtml = badge.outerHTML.replace(badgeText, highlightText(badgeText, searchText));
                            daysCell.innerHTML = badgeHtml;
                        } else {
                            daysCell.innerHTML = highlightText(daysCell.textContent, searchText);
                        }
                    }
                });
                
                // Nincs találat üzenet megjelenítése
                const container = document.getElementById('tab-expiring-container');
                let noResultsMessage = container.querySelector('.no-results');
                
                if (!hasVisibleRows && searchText !== '') {
                    if (!noResultsMessage) {
                        noResultsMessage = document.createElement('div');
                        noResultsMessage.className = 'no-results';
                        noResultsMessage.textContent = 'Nincs találat a keresési feltételeknek megfelelően';
                        container.appendChild(noResultsMessage);
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
    }, 50);
}

async function showEditCardForm(cardId) {
    try {
        const response = await fetch(`/api/cards/${cardId}`, {
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            throw new Error('Nem sikerült a kártya adatainak lekérése');
        }

        const card = await response.json();

        // Fetch users for the dropdown
        await fetchUsers();

        const expiryDate = card.expiry_date ? new Date(card.expiry_date).toISOString().split('T')[0] : '';

        // Szűrjük ki azokat a felhasználókat, akiknek már van kártyájuk, kivéve a jelenlegi kártya tulajdonosát
        const usersWithCards = cards
            .filter(c => c.id !== card.id)  // Kihagyjuk a jelenlegi kártyát
            .map(c => c.user_id);

        // Az elérhető felhasználók: akiknek nincs más kártyájuk, vagy ők a jelenlegi kártya tulajdonosai
        const availableUsers = users.filter(user =>
            !usersWithCards.includes(user.id) || user.id === card.user_id
        );

        const cardForm = `
            <form id="edit-card-form">
                <div class="form-group">
                    <label for="edit-card-id">Kártya azonosító</label>
                    <input type="text" id="edit-card-id" name="card-id" value="${card.card_id}" readonly>
                    <div class="form-hint">A kártya azonosítója nem módosítható</div>
                </div>
                <div class="form-group">
                    <label for="edit-user-select">Felhasználó</label>
                    <select id="edit-user-select" name="user-select">
                        <option value="">Válasszon felhasználót</option>
                        ${availableUsers.map(user => `<option value="${user.id}" ${user.id === card.user_id ? 'selected' : ''}>${user.first_name} ${user.last_name}</option>`).join('')}
                    </select>
                    <div class="form-hint">Csak olyan felhasználók jelennek meg, akiknek nincs más kártyájuk</div>
                </div>
                <div class="form-group">
                    <label for="edit-expiry-date">Lejárati dátum (opcionális)</label>
                    <input type="date" id="edit-expiry-date" name="expiry-date" value="${expiryDate}">
                    <div class="form-hint">Ha üresen hagyja, a kártya nem jár le</div>
                </div>
                <div class="form-group">
                    <label for="edit-status">Státusz</label>
                    <select id="edit-status" name="status">
                        <option value="active" ${card.status === 'active' ? 'selected' : ''}>Aktív</option>
                        <option value="blocked" ${card.status === 'blocked' ? 'selected' : ''}>Blokkolt</option>
                        <option value="revoked" ${card.status === 'revoked' ? 'selected' : ''}>Visszavont</option>
                    </select>
                </div>
            </form>
        `;

        const handleConfirm = async () => {
            const userId = document.getElementById('edit-user-select').value;
            const expiryDateElement = document.getElementById('edit-expiry-date');
            const expiryDate = expiryDateElement.value ? new Date(expiryDateElement.value) : null;
            const status = document.getElementById('edit-status').value;

            try {
                const response = await fetch(`/api/cards/${cardId}`, {
                    method: 'PUT',
                    headers: {
                        'Content-Type': 'application/json',
                        'Authorization': `Bearer ${authToken}`
                    },
                    body: JSON.stringify({
                        user_id: userId ? Number(userId) : null,
                        expiry_date: expiryDate,
                        status: status
                    })
                });

                if (!response.ok) {
                    const errorData = await response.json();
                    throw new Error(errorData.error || 'Nem sikerült a kártya frissítése');
                }

                alert('Kártya sikeresen frissítve!');
                hideModal();
                await fetchCards();
                showCardManagementModal(new Event('click'));
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Kártya szerkesztése', cardForm, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

function showCardRegistrationModal(e) {
    e.preventDefault();

    // Ellenőrizzük, hogy vannak-e felhasználók a listában
    if (!users || users.length === 0) {
        alert('Nincsenek felhasználók a rendszerben. Először hozzon létre felhasználót!');
        return;
    }

    // Szűrjük ki azokat a felhasználókat, akiknek már van kártyájuk
    const usersWithCards = cards.map(card => card.user_id);
    const availableUsers = users.filter(user => !usersWithCards.includes(user.id));

    if (availableUsers.length === 0) {
        alert('Minden felhasználónak már van kártyája. Töröljön vagy módosítson meglévő kártyákat, vagy hozzon létre új felhasználót.');
        return;
    }

    const modalContent = `
        <form id="card-form">
            <div class="form-group">
                <label for="card-id">Kártya azonosító</label>
                <input type="text" id="card-id" name="card-id" required placeholder="Adja meg a kártya azonosítóját">
                <div class="form-hint">A kártya egyedi azonosítója, amely az RFID chipben található</div>
            </div>
            <div class="form-group">
                <label for="user-select">Felhasználó</label>
                <select id="user-select" name="user-select" required>
                    <option value="">Válasszon felhasználót</option>
                    ${availableUsers.map(user => `<option value="${user.id}">${user.first_name} ${user.last_name}</option>`).join('')}
                </select>
                <div class="form-hint">Válassza ki, melyik felhasználóhoz rendelje a kártyát (csak olyan felhasználók jelennek meg, akiknek még nincs kártyájuk)</div>
            </div>
            <div class="form-group">
                <label for="expiry-date">Lejárati dátum (opcionális)</label>
                <input type="date" id="expiry-date" name="expiry-date">
                <div class="form-hint">Ha üresen hagyja, a kártya nem jár le</div>
            </div>
        </form>
    `;

    const handleConfirm = async () => {
        const cardId = document.getElementById('card-id').value;
        const userId = document.getElementById('user-select').value;
        const expiryDateElement = document.getElementById('expiry-date');
        const expiryDate = expiryDateElement.value ? new Date(expiryDateElement.value) : null;

        // Részletes hibaellenőrzés
        if (!cardId) {
            alert('Kérjük, adja meg a kártya azonosítóját!');
            return;
        }

        if (!userId) {
            alert('Kérjük, válasszon felhasználót a listából!');
            return;
        }

        try {
            const response = await fetch('/api/cards', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Authorization': `Bearer ${authToken}`
                },
                body: JSON.stringify({
                    card_id: cardId,
                    user_id: Number(userId),
                    expiry_date: expiryDate
                })
            });

            if (!response.ok) {
                const errorData = await response.json();
                throw new Error(errorData.error || 'Nem sikerült a kártya regisztrációja');
            }

            alert('Kártya sikeresen regisztrálva!');
            hideModal();
            await fetchCards();
            showCardManagementModal(new Event('click'));
        } catch (error) {
            alert('Hiba történt: ' + error.message);
        }
    };

    showModal('Kártya regisztrálása', modalContent, handleConfirm);
}

async function deleteCard(cardId) {
    try {
        const response = await fetch(`/api/cards/${cardId}`, {
            method: 'DELETE',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a kártya törlése');
        }

        alert('Kártya sikeresen törölve!');
        await fetchCards();
        showCardManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function blockCard(cardId) {
    try {
        const response = await fetch(`/api/cards/${cardId}/block`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a kártya blokkolása');
        }

        alert('Kártya sikeresen blokkolva!');
        await fetchCards();
        showCardManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function unblockCard(cardId) {
    try {
        const response = await fetch(`/api/cards/${cardId}/unblock`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a kártya feloldása');
        }

        alert('Kártya sikeresen feloldva!');
        await fetchCards();
        showCardManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

async function revokeCard(cardId) {
    try {
        const response = await fetch(`/api/cards/${cardId}/revoke`, {
            method: 'POST',
            headers: {
                'Authorization': `Bearer ${authToken}`
            }
        });

        if (!response.ok) {
            const errorData = await response.json();
            throw new Error(errorData.error || 'Nem sikerült a kártya visszavonása');
        }

        alert('Kártya sikeresen visszavonva!');
        await fetchCards();
        showCardManagementModal(new Event('click'));
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}

function getCardStatusBadge(status) {
    switch (status) {
        case 'active': return 'badge-success';
        case 'blocked': return 'badge-warning';
        case 'revoked': return 'badge-error';
        case 'expired': return 'badge-error';
        case 'pending': return 'badge-primary';
        default: return 'badge-primary';
    }
}

function translateCardStatus(status) {
    const statusMap = {
        'active': 'Aktív',
        'blocked': 'Blokkolt',
        'revoked': 'Visszavont',
        'expired': 'Lejárt',
        'pending': 'Függőben'
    };
    return statusMap[status] || status;
}

// Lejáró kártyák tab betöltése
async function loadExpiringCardsTab() {
    const container = document.getElementById('tab-expiring-container');
    if (!container) return;

    container.innerHTML = '<p class="loading-message">Lejáró kártyák betöltése...</p>';

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

        if (!expiringCards || expiringCards.length === 0) {
            container.innerHTML = '<p class="info-message">Nincsenek hamarosan lejáró kártyák</p>';
            return;
        }

        // Rendezzük a kártyákat a lejárat dátuma szerint
        expiringCards.sort((a, b) => new Date(a.expiry_date) - new Date(b.expiry_date));

        // Táblázat létrehozása
        let tableHTML = `
            <div class="table-responsive">
                <table id="expiring-cards-table">
                    <thead>
                        <tr>
                            <th>Felhasználó</th>
                            <th>Kártya azonosító</th>
                            <th>Lejárati dátum</th>
                            <th>Hátralevő idő</th>
                            <th>Műveletek</th>
                        </tr>
                    </thead>
                    <tbody>
            `;

        expiringCards.forEach(card => {
            const expiryDate = new Date(card.expiry_date);
            const today = new Date();
            const daysRemaining = Math.ceil((expiryDate - today) / (1000 * 60 * 60 * 24));

            const userName = card.user ? `${card.user.first_name} ${card.user.last_name}` : 'Ismeretlen';

            // Osztályozás a hátralévő napok száma alapján
            let rowClass = '';
            if (daysRemaining <= 7) {
                rowClass = 'urgent-row';
            } else if (daysRemaining <= 30) {
                rowClass = 'warning-row';
            }

            tableHTML += `
                <tr class="${rowClass}">
                    <td>${userName}</td>
                    <td>${card.card_id}</td>
                    <td>${expiryDate.toLocaleDateString('hu-HU')}</td>
                    <td><span class="badge ${daysRemaining <= 7 ? 'badge-error' : 'badge-warning'}">${daysRemaining} nap</span></td>
                    <td>
                        <button class="extend-card-tab" data-id="${card.id}">Hosszabbítás</button>
                        <button class="edit-card" data-id="${card.id}">Szerkesztés</button>
                        ${card.status === 'active' ? `<button class="block-card" data-id="${card.id}">Zárolás</button>` : ''}
                    </td>
                </tr>
            `;
        });

        tableHTML += `</tbody></table></div>`;

        container.innerHTML = tableHTML;

        // Eseménykezelők hozzáadása
        document.querySelectorAll('.extend-card-tab').forEach(button => {
            button.addEventListener('click', () => {
                const cardId = button.getAttribute('data-id');
                showExtendCardForm(cardId);
            });
        });

        document.querySelectorAll('#tab-expiring-container .edit-card').forEach(button => {
            button.addEventListener('click', () => {
                const cardId = button.getAttribute('data-id');
                showEditCardForm(cardId);
            });
        });

        document.querySelectorAll('#tab-expiring-container .block-card').forEach(button => {
            button.addEventListener('click', async () => {
                const cardId = button.getAttribute('data-id');
                if (confirm('Biztosan zárolni szeretné ezt a kártyát?')) {
                    await blockCard(cardId);
                }
            });
        });
        
        // Ellenőrizzük a scrollbart, amikor a táblázat beillesztve
        // Két időpontban ellenőrizzük: azonnal, és kis késleltetéssel is (tartalmi betöltés után)
        if (typeof checkScrollbars === 'function') {
            checkScrollbars();
            setTimeout(checkScrollbars, 100);
        }
        
        // Aktiváljuk a keresést az új betöltött táblázatra
        const searchInput = document.getElementById('expiring-card-search');
        if (searchInput && searchInput.value) {
            // Ha már van keresési érték, alkalmazzuk azt
            searchInput.dispatchEvent(new Event('input'));
        }

    } catch (error) {
        // Console.error eltávolítva
        container.innerHTML = '<p class="error-message">Hiba történt a lejáró kártyák betöltése közben.</p>';
    }
}

// Kártya hosszabbítás űrlap megjelenítése
async function showExtendCardForm(cardId) {
    try {
        // Kártya adatainak betöltése
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

        // Alapértelmezett új lejárat: 1 év a mostani dátumtól
        const defaultNewExpiry = new Date();
        defaultNewExpiry.setFullYear(defaultNewExpiry.getFullYear() + 1);

        // Formátum konverzió az input mezőhöz (YYYY-MM-DD)
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

                // Frissítsük a kártyákat és a lejáró kártyák listáját
                await Promise.all([
                    fetchCards(),
                    loadExpiringCardsTab()
                ]);

                // Frissítsük a dashboard-ot is, ha van rajta lejáró kártyák lista
                if (typeof fetchExpiringCards === 'function') {
                    fetchExpiringCards();
                }

                // Frissítsük a kártyák kezelése modalt
                showCardManagementModal();
            } catch (error) {
                alert('Hiba történt: ' + error.message);
            }
        };

        showModal('Kártya hosszabbítása', formContent, handleConfirm);
    } catch (error) {
        alert('Hiba történt: ' + error.message);
    }
}