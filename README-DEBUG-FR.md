# Debugger le code métier Fastly Compute nativement

Fastly Compute exécute votre code dans le runtime WebAssembly de Fastly. Ce runtime est nécessaire pour les fonctionnalités edge, mais ce n'est pas le meilleur endroit pour debugger de la logique métier classique.

Ce document explique comment garder le même code métier et ajouter un adaptateur de runtime natif pour :

- lancer l'application localement sans Fastly/Viceroy ;
- utiliser un debugger normal et des breakpoints ;
- écrire des tests unitaires/intégration rapides ;
- tester le routing, les status codes, les headers, la query et le proxy backend.

Le runtime natif sert à debugger la logique métier. Le comportement du cache edge doit toujours être validé avec `fastly compute serve` ou sur Fastly.

---

## Par où commencer

Partez de l'entrypoint Fastly, pas d'un nouveau serveur natif.

1. Identifiez l'entrypoint Fastly et listez chaque type/fonction SDK Fastly utilisé.
2. Séparez les règles métier du runtime :
   - routes ;
   - règles de méthode ;
   - valeurs TTL/SWR ;
   - construction de l'URL backend ;
   - status codes ;
   - headers ;
   - réponses d'erreur.
3. Déplacez ces règles dans une application layer qui n'importe pas le SDK Fastly.
4. Gardez l'entrypoint Fastly fonctionnel via un adaptateur Fastly.
5. Ajoutez un adaptateur natif et un entrypoint natif qui appellent la même application layer.

Le runtime natif ne doit pas changer le comportement métier. Il remplace seulement le runtime hôte.

---

## Les 3 étapes

### Étape 1 — DDD : séparer la logique métier

But : isoler le code applicatif indépendant du runtime.

Actions :

- Déplacer l'entrypoint Fastly dans un fichier ou binaire dédié.
- Extraire le routing, les constantes backend, la cache policy et les règles de response dans une application layer.
- Garder l'entrypoint Fastly comme un wrapper hôte fin.
- Mettre à jour la configuration de build Fastly si le path de l'entrypoint change.
- Garder le comportement Fastly existant inchangé.

Règles :

- L'application layer ne doit pas importer les types du SDK Fastly.
- L'application layer ne doit pas créer de request ou response Fastly.
- L'application layer doit être testable sans runtime Fastly.

Résultat attendu :

```text
application/
  backend.*      # constantes, logique de proxy, types request/response neutres
  handler.*      # routing, vérification de méthode, status codes
```

L'entrypoint Fastly fonctionne toujours exactement comme avant.

---

### Étape 2 — Interfaces/Ports : définir la frontière du runtime

But : définir le contrat minimum dont le code métier a besoin, quel que soit le runtime.

Ports typiques :

| Port | Responsabilité |
|---|---|
| Request | méthode, path, raw query string |
| Response | status, headers, body |
| Header collection | set/add/get/delete/iterate headers |
| Backend client | créer et envoyer une request upstream |
| Cache options | valeurs TTL et stale-while-revalidate |
| Response writer | écrire status, headers, body vers l'hôte |

Règles :

- Les ports doivent décrire le besoin métier, pas le SDK Fastly.
- L'application layer dépend uniquement des ports.
- Le code spécifique au runtime implémente les ports.
- Les cache options peuvent traverser le port même si un runtime natif les ignore.

Ensuite, implémentez l'adaptateur Fastly :

- wrapper les request/response Fastly ;
- envoyer vers le backend nommé ;
- appliquer TTL/SWR ;
- copier headers, status et body.

À la fin de cette étape :

- l'application layer contient la logique métier ;
- l'adaptateur Fastly contient le comportement spécifique Fastly ;
- l'entrypoint Fastly ne fait que câbler l'adaptateur et démarrer le runtime.

---

### Étape 3 — Dependency Injection : ajouter le runtime natif

But : exécuter la même application layer via un adaptateur natif.

Actions :

- implémenter un backend client natif ;
- implémenter un entrypoint de serveur HTTP natif ;
- injecter l'adaptateur natif dans le composition root natif ;
- injecter l'adaptateur Fastly dans le composition root Fastly ;
- écouter sur un port local comme `:8080` ;
- ignorer les cache options edge Fastly dans l'adaptateur natif, sauf si vous ajoutez une couche de cache séparée.

Les composition roots sont volontairement minuscules :

```text
Entrypoint Fastly :
  setClient(FastlyAdapter)
  start Fastly handler

Entrypoint natif :
  setClient(NativeAdapter)
  listen on :8080
  call application.handle(request)
```

Résultat attendu :

```text
application/          # logique métier, sans SDK runtime
fastly-adapter/       # implémentation SDK Fastly
native-adapter/       # implémentation HTTP native
fastly-entrypoint/    # composition root Fastly
native-entrypoint/    # composition root natif
```

Workflow de debug :

```bash
# Comportement Fastly et vérifications de cache
fastly compute serve --watch

# Debug natif de la logique métier
run-native-server
curl -i http://localhost:8080/current?country=FR&city=Lyon
```

Utilisez les breakpoints et les logs dans l'application layer, pas dans l'adaptateur, pour debugger les règles métier.

---

## Mapping runtime par langage

### Go

| Concern | SDK Fastly | Go natif |
|---|---|---|
| Entrypoint HTTP | `fsthttp.ServeFunc` | `net/http.Server` + `http.ServeMux` |
| Request entrante | `*fsthttp.Request` | `*http.Request` |
| Response writer | `fsthttp.ResponseWriter` | `http.ResponseWriter` |
| Headers | `fsthttp.Header` | `http.Header` |
| Request backend | `fsthttp.NewRequest` | `http.NewRequestWithContext` |
| Envoi au backend | `(*fsthttp.Request).Send(ctx, "backend_api")` | `http.Client.Do` |
| Cache options edge | `fsthttp.CacheOptions{TTL, StaleWhileRevalidate}` | ignorées par l'adaptateur natif |
| Response d'erreur | `fsthttp.Error` / `fsthttp.NotFound` | `http.Error` / `WriteHeader` |
| DI | `application.SetClient(fastlyadapter.Client{})` | `application.SetClient(httpadapter.Client{})` |
| Entrypoint Fastly | `cmd/fastly/main.go` | — |
| Entrypoint natif | — | `cmd/http/main.go` |

### JavaScript

| Concern | SDK Fastly | JavaScript natif |
|---|---|---|
| Entrypoint HTTP | `addEventListener("fetch", ...)` + `event.respondWith` | `node:http.createServer` |
| Request entrante | `event.request` (`Request`) | `node:http` `IncomingMessage` |
| Response | `Response` | `node:http` `ServerResponse` |
| Headers | `Headers` | `rawHeaders` / `setHeader` |
| Request backend | `fetch(url, { backend: "backend_api" })` | `node:https.request` |
| Envoi au backend | `fetch` | `https.request` + événements `response` |
| Cache options edge | `CacheOverride("override", { ttl, swr })` | ignorées par l'adaptateur natif |
| Response d'erreur | `new Response(status)` | `res.statusCode` + `res.end` |
| DI | `setClient(new FastlyClient())` | `setClient(new NativeClient())` |
| Entrypoint Fastly | `src/fastly.js` | — |
| Entrypoint natif | — | `src/http.js` |

### Rust

| Concern | SDK Fastly | Rust natif |
|---|---|---|
| Entrypoint HTTP | `#[fastly::main]` | `std::net::TcpListener` + boucle HTTP |
| Request entrante | `fastly::Request` | `application::Request` depuis `TcpStream` |
| Response | `fastly::Response` | `application::Response` écrit sur `TcpStream` |
| Headers | méthodes header request/response Fastly | builder `ureq` / headers HTTP bruts |
| Request backend | `fastly::Request::get(url)` | `ureq::Agent::request` |
| Envoi au backend | `.send("backend_api")` | `.call()` |
| Cache options edge | `.with_ttl(...)` / `.with_stale_while_revalidate(...)` | ignorées par l'adaptateur natif |
| Response d'erreur | `Response::from_status(...)` | `Response { status, ... }` neutre |
| DI | `application::set_client(Box::new(FastlyClient))` | `application::set_client(Box::new(NativeClient))` |
| Entrypoint Fastly | `src/bin/fastly.rs` | — |
| Entrypoint natif | — | `src/bin/http.rs` |

Important : les adaptateurs natifs ne reproduisent pas le cache edge Fastly. Testez TTL, SWR, cache hits et comportement surrogate avec Fastly.

---

## Contrat métier partagé

Les trois implémentations gardent les mêmes règles :

| Path | TTL | SWR |
|---|---:|---:|
| `/` | 3600 | 7200 |
| `/current` | 900 | 3600 |
| `/next` | 900 | 3600 |
| `/city` | 60 | 300 |

Comportement partagé :

- uniquement `GET` et `HEAD` ;
- les autres méthodes renvoient `405` avec `Allow: GET, HEAD` ;
- les paths inconnus renvoient `404` ;
- les query strings sont préservées ;
- les requests backend utilisent `backend_api` ;
- l'hôte upstream est `nominally-gentle-rodent.edgecompute.app` ;
- les headers, le status et le body de response sont transmis.

---

## Solutions par branche

Branche de base : `solution-backend`.

### Go

| Étape | Branche |
|---|---|
| Déplacer l'entrypoint Fastly | `go/step1` |
| Ports + adaptateur Fastly + DI | `go/step2` |
| Adaptateur natif `net/http` | `go/step3` |

Lancer le Go natif :

```bash
git checkout go/step3
cd go
go run ./cmd/http
```

### JavaScript

| Étape | Branche |
|---|---|
| Déplacer l'entrypoint Fastly | `js/step1` |
| Ports + adaptateur Fastly + DI | `js/step2` |
| Adaptateur Node HTTP natif | `js/step3` |

Lancer le JavaScript natif :

```bash
git checkout js/step3
cd js
npm run start:http
```

Avec Bun :

```bash
bun ./src/http.js
```

### Rust

| Étape | Branche |
|---|---|
| Déplacer l'entrypoint Fastly | `rust/step1` |
| Ports + adaptateur Fastly + DI | `rust/step2` |
| Adaptateur HTTP natif | `rust/step3` |

Lancer le Rust natif :

```bash
git checkout rust/step3
cd rust
cargo run --features native-http --bin http
```

Tous les serveurs natifs écoutent sur `127.0.0.1:8080`.

---

## Boucle de debug recommandée

1. Démarrez le serveur natif du langage utilisé.
2. Posez des breakpoints dans `application/` ou `internal/application/`.
3. Appelez l'endpoint avec `curl`.
4. Corrigez la logique métier.
5. Relancez les tests natifs.
6. Validez le comportement spécifique Fastly avec `fastly compute serve --watch`.
7. L'adaptateur Fastly doit rester simple; il ne doit pas contenir de règles métier.
