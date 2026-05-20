# 実験2: PI制御による目標速度ステップ追従

## 概要

実験1で同定した1次系モデル $\omega(s)/I(s) = K/(\tau s + 1)$ に対して、AtomS3 上で離散時間 PI 制御

$$
u = K_p\, e + K_i \int e\, dt, \quad e = \omega_{\text{ref}} - \omega
$$

を実装し、目標角速度 $\omega_{\text{ref}}$ をステップ的に変えたときの追従特性を計測する。制御出力 $u$ は電流コマンド [mA] として Roller485 に送られる。

理論背景・ゲイン設計指針は [`../plan.md`](../plan.md) を参照（実験2節は今後埋める予定）。

## ディレクトリ構成

- `5me-me2-ce-pi.ino` — AtomS3 ファームウェア (PI 制御ループを含む)
- `roller485-pi/` — PC ホストツール (Go)

## ハードウェア

- M5Stack AtomS3 + ATOM MOTION + Unit-Roller485 (I2C)
- 接続: ATOM MOTION PORT C (`SDA=6, SCL=5`)。AtomS3 本体 Grove (PORT A, `SDA=2, SCL=1`) を使う場合はファームウェア冒頭のピン定義をコメント差し替え。

## シリアルプロトコル

- ボーレート: 115200
- コマンド形式 (改行で送信):

  ```
  Kp Ki omega_pre[RPM] omega_step[RPM] T_hold[ms]
  ```

  例: `0.5 2.0 100 300 5000` → Kp=0.5, Ki=2.0 で 100 RPM に 5000 ms 整定 → 300 RPM へ目標値ステップして 5000 ms 計測。
- 計測中 `stop\n` を送るとモータ停止 + 中断。
- CSV (シリアル → ホスト):

  ```
  time_ms,omega_ref_rpm,speed_rpm,current_cmd_ma,current_meas_ma
  0,100.0,0.0,500.0,498.3
  20,100.0,15.2,500.0,495.1
  ...
  ```

  サンプリング周期は 20 ms。`current_cmd_ma` は PI が計算した電流コマンド、`current_meas_ma` は Roller485 のリードバック値。

## 制御仕様

- 制御周期: 20 ms (`SAMPLE_MS`)
- 電流飽和: ±1000 mA (`I_MAX_MA`)
- アンチワインドアップ: 積分クランプ式 (`integral` を $\pm I_{\max} / |K_i|$ に制限)
- $K_i = 0$ のときはクランプをスキップして純 P 制御として動く

## 使い方

1. ファームウェア書き込み: Arduino IDE で `5me-me2-ce-pi.ino` を AtomS3 に書き込む。
2. ホスト起動:

   ```bash
   cd roller485-pi
   go run .
   ```

   または `task` でクロスビルドした `dist/` 配下のバイナリを実行。
3. シリアルポート選択 → `Kp` / `Ki` / `omega_pre [RPM]` / `omega_step [RPM]` / `T_hold [ms]` を入力。
4. 実行ファイルと同じディレクトリに `YYYYMMDDhhmmss.csv` と `.png` が自動保存される。`Ctrl-C` または `stop` 入力で中断可能。

## ビルド (ホスト)

```bash
cd roller485-pi
go run .                       # 開発実行 (現プラットフォーム向け)
task                           # Win/Linux/macOS 同時クロスビルド → dist/
task windows / linux / macos   # プラットフォーム個別
```

Go 1.25 系が必要 (`go.mod` の `go 1.25.9` 指定)。

## 出力

- `YYYYMMDDhhmmss.csv` — 計測データ。1行目にヘッダ `time_ms,omega_ref_rpm,speed_rpm,current_cmd_ma,current_meas_ma` が残る（ホスト側は数値パース失敗行を黙って捨てる設計）。
- `YYYYMMDDhhmmss.png` — プロット。
  - 上半分: `omega_ref` (灰) と速度 (青) を同スケールで重ね描き → 追従誤差が一目で見える
  - 下半分: `current_cmd` (橙) と `current_meas` (赤) を同スケールで重ね描き → コマンドと実測の乖離が見える
  - 軸ラベル無し、自動スケール。

## ゲイン設計

`Kp` `Ki` の初期値の選び方、追従特性の評価指標 (オーバーシュート、整定時間、IAE など) は [`../plan.md`](../plan.md) 実験2節を参照（今後埋める）。
