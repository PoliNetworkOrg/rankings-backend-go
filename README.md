# [WIP] Rankings Backend (Go)

## Rankings Project Structure
- Frontend (React/ViteJS): 
    - Live: https://rankings.polinetwork.org/ 
    - Repo: https://github.com/polinetworkorg/Rankings

- Backend: (newer is first)
    - Go: this repo (WIP)
    - **C#: https://github.com/PoliNetworkOrg/GraduatorieScriptCSharp (stable)**
    - *Py: https://github.com/PoliNetworkOrg/GraduatorieScript (deprecated)*

- Data: https://github.com/PoliNetworkOrg/RankingsDati


## Usage

> [!NOTE]  
> The following instructions should be stable, but if they are not working anymore, please open an issue.

There are 4 total commands (ATTOW):
- `scraper`: perform scraping of rankings html files and school manifests against Polimi website
- `parser`: perform parsing of raw html files into custom data shapes, output as JSON 
- `playground`: for testing purpose only, especially useful when dealing with JSON encoding/decoding. Do not expect this package to last forever.
- `migrate`: made to convert old `html` folder structure to the new one. See [`2b99e43`](https://github.com/PoliNetworkOrg/rankings-backend-go/commit/2b99e43925cd3435a5b7a0fb4bb4911c1d085ff1),
[`3f57469`](https://github.com/PoliNetworkOrg/rankings-backend-go/commit/3f57469cab785f89d7dd93451b73f427e3fd33a1),
[`9008f83`](https://github.com/PoliNetworkOrg/rankings-backend-go/commit/9008f83e4e1710f27f3a8bdcc6f47a45c82f1ecc)
commits for more details

The most common scenario is the following:
1. Run scraper
    ```bash
    go run ./cmd/scraper -d ../RankingsDati/data
    ```
2. Run parser
    ```bash
    go run ./cmd/parser -d ../RankingsDati/data
    ```
> [!IMPORTANT]  
> If you don’t provide the `-d` (`--data-dir`) argument to either the scraper or parser commands, they will default to using the temporary folder `./tmp`.  
> To understand why we are passing a `data` folder from another repository, check [the C# README](https://github.com/PoliNetworkOrg/GraduatorieScriptCSharp?tab=readme-ov-file#data-folder).  
> Note that for the purpose of using this script, it is possible to use a folder inside this project (e.g. `./data`), but it is not recommended.    

You can change the log level with the `LOG_LEVEL` env variable (`debug`/`info`/`warn`/`error`). Example:
```bash
LOG_LEVEL=error go run ./cmd/parser
```
