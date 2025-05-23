# 叢集

自 v2.0.0-beta.23 起，您可以在設定檔的 `cluster` 分區中定義多個環境。

## Node

- 版本：`>= v2.0.0-beta.23`
- 類型：`string`
- 結構：`Scheme://Host(:Port)?name=環境名稱&node_secret=節點金鑰&enabled=是否啟用`
- 範例：`http://10.0.0.1:9000?name=node1&node_secret=my-node-secret&enabled=true`


如果您需要設定多個環境，請參考下面的設定：
```ini
[cluster]
Node = http://10.0.0.1:9000?name=node1&node_secret=my-node-secret&enabled=true
Node = http://10.0.0.2:9000?name=node2&node_secret=my-node-secret&enabled=false
Node = http://10.0.0.3?name=node3&node_secret=my-node-secret&enabled=true
```

預設情況下，Nginx UI 將在啟動階段執行環境的建立操作，您也可以在 WebUI 中的環境列表中找到「從設定中載入」按鈕，手動更新環境。

為了避免與資料庫內已經存在的環境衝突，Nginx UI 會檢查 Scheme://Host(:Port) 部分是否應是否唯一，
如果不存在，則按照設定進行建立，反之則不會進行任何操作。

注意：如果您刪除了設定檔中的某個節點，Nginx UI 不會刪除資料庫中的記錄。
