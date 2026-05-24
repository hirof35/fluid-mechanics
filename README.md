2D Fluid Dynamics Simulation in GoGo言語（Golang）と超軽量2Dゲームライブラリ Ebitengine を使用した、リアルタイム2D流体物理シミュレーションのインタラクティブデモです。
ゲーム開発やシミュレーションプロトタイピングにおける流体表現のベースとして利用できます。
<img width="637" height="671" alt="スクリーンショット 2026-05-24 161533" src="https://github.com/user-attachments/assets/9f906057-6111-4036-971e-8c349a0a1c18" />

🚀 特徴リアルタイム演算: Jos Stam氏の有名な論文 「Real-Time Fluid Dynamics for Games」 のアルゴリズム（安定的半物質移動法）をベースに実装。
ナビエ・ストークス方程式の近似: 拡散（Diffuse）、移流（Advect）に加え、質量保存則を満たすための投影ステップ（Project）を実装しているため、リアルな「渦（Vortex）」が巻く様子を観察できます。
インタラクティブ性: マウスの左クリックおよびドラッグ操作により、流体（密度場）の注入と外力の付加をリアルタイムに行えます。
高速描画: 各格子のデータをRGBAピクセルバッファ（[]byte）へダイレクトに書き込み、GPUへ一括転送（WritePixels）することで、60fpsの滑らかな動作を実現しています。
🛠 動作環境Go 1.18 以上OS: Windows, macOS, Linux (Ebitengine の動作要件に準拠)
📦 インストールと実行方法リポジトリのクローン（またはディレクトリの作成）Bash
mkdir fluid-mechanics
cd fluid-mechanics

2. **Goモジュールの初期化**
   ```bash
go mod init fluid-mechanics
Ebitengineのインストールgo get github.com/hajimehoshi/ebiten/v2
4. **シミュレーションの実行**
   ```bash
go run main.go
🎮 操作方法マウス左クリック / ドラッグ: 画面上をクリックまたはドラッグすると、その位置に「煙（密度）」が生成され、同時に左上から右下方向への速度ベクトルが加わります。
流体は時間の経過とともに周囲の速度場に流され、徐々に減衰して消滅します。
📐 アルゴリズム概要本プログラムは空間を $N \times N$ の不連続な格子（グリッド）に分割して計算を行う格子法を採用しています。
毎フレーム（$1/60$ 秒）、以下のステップで流体の方程式を解いています
：$$\frac{\partial \mathbf{u}}{\partial t} + (\mathbf{u} \cdot \nabla)\mathbf{u} = \nu \nabla^2 \mathbf{u} + \mathbf{f}$$速度場の更新 (Velocity Solver)外力 (Add Source): マウス等による入力ベクトルの追加。
拡散 (Diffuse): 粘性係数（visc）に従い、速度を周囲に分散（ガウス＝ザイデル法の反復解法）。
投影 (Project): 質量保存（非圧縮性条件 $\nabla \cdot \mathbf{u} = 0$）を満たすため、ポアソン方程式を解いて湧き出し・吸い込み成分を除去。
これにより綺麗な渦が形成されます。移流 (Advect): 速度場自身によって、速度そのものが運ばれる現象をセミ・ラグランジアン法で計算。密度場の更新 (Density Solver)拡散 (Diffuse): 煙の拡散係数（diff）に従い、周囲ににじむように広げる。
移流 (Advect): 更新された速度場に沿って、煙の密度を移動させる。
📄 ライセンスMIT License
