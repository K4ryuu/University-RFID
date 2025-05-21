# Egyetemi RFID Beléptető Rendszer

Ez a projekt egy egyetemi feladatként készült RFID-alapú beléptető rendszer, amely lehetővé teszi az egyetemi helyiségekhez való hozzáférés szabályozását RFID kártyák segítségével.

## A projekt célja

Az RFID Beléptető Rendszer egy olyan megoldás, amely:

- Lehetővé teszi a felhasználók számára az egyetemi termekbe történő belépést RFID kártyák használatával
- Biztosítja a termekhez való hozzáférés jogosultságainak kezelését
- Naplózza a belépési eseményeket
- Csoportalapú hozzáférés-kezelést tesz lehetővé
- Valós idejű monitorozást biztosít websocket kapcsolaton keresztül

## Funkciók

- **Felhasználókezelés**: Felhasználók regisztrálása, módosítása, törlése
- **Kártyakezelés**: RFID kártyák hozzárendelése felhasználókhoz
- **Teremkezelés**: Termek felvétele, jogosultságok beállítása
- **Csoportkezelés**: Felhasználók csoportokba szervezése a könnyebb jogosultságkezelés érdekében
- **Jogosultságkezelés**: Ki, mikor, mely termekbe léphet be
- **Eseménynaplózás**: Minden belépési kísérlet rögzítése
- **Valós idejű értesítések**: Websocket kapcsolaton keresztül
- **Szimuláció**: Virtuális RFID olvasók szimulálása tesztelési célokra

## Technológiák

- **Backend**: Go nyelv, Gorilla WebSocket
- **Frontend**: HTML, CSS, JavaScript
- **Adatbázis**: Konfigurálható (alapértelmezetten SQLite)
- **Biztonság**: JWT token alapú autentikáció, API-kulcs védelem
- **RFID**: RFID olvasók integrációja

## Telepítés és Használat

1. Klónozd a repót: `git clone https://github.com/K4ryuu/University-RFID.git`
2. Navigálj a projekt mappájába: `cd University-RFID`
3. Másold le és nevezd át a `.env.example` fájlt `.env`-re a `src` mappában
4. Állítsd be a környezeti változókat a `.env` fájlban
5. Indítsd el a szervert: `cd src && go run cmd/rfid-server/main.go`
6. Nyisd meg a böngészőt a `http://localhost:8080` címen (vagy amin beállítottad)

## Fejlesztői Dokumentáció

A projekt struktúrája:

- `src/cmd/rfid-server`: A fő belépési pont
- `src/internal/models`: Adatmodellek
- `src/internal/handlers`: HTTP kérések kezelői
- `src/internal/middleware`: Middleware-ek (autentikáció, stb.)
- `src/internal/routes`: API útvonalak definíciói
- `src/internal/utils`: Segédfunkciók
- `src/internal/websocket`: Valós idejű kommunikáció
- `src/web`: Frontend fájlok (HTML, CSS, JS)

## Egyetemi Projekt Információ

Ez a projekt oktatási célokra készült, amely demonstrálja a modern webalkalmazások fejlesztését, az RFID technológia integrációját, valamint a hozzáférés-vezérlési rendszerek működését.

## Licensz

Ez a projekt az MIT licensz alatt áll. Lásd a [LICENSE](LICENSE) fájlt a részletekért.
