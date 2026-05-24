package main

import (
	"log"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	N          = 64                // 格子サイズ (64x64)
	size       = (N + 2) * (N + 2) // 境界を含めた配列サイズ
	scale      = 8                 // 1格子あたりの描画ピクセルサイズ
	screenWidth  = N * scale       // ウィンドウの幅 (512px)
	screenHeight = N * scale       // ウィンドウの高さ (512px)
)

type Game struct {
	u    []float64 // X方向速度
	v    []float64 // Y方向速度
	uOld []float64
	vOld []float64
	d    []float64 // 密度
	dOld []float64

	// Ebitengine描画用のピクセルバッファ (RGBA)
	pix []byte
}

func NewGame() *Game {
	return &Game{
		u:    make([]float64, size),
		v:    make([]float64, size),
		uOld: make([]float64, size),
		vOld: make([]float64, size),
		d:    make([]float64, size),
		dOld: make([]float64, size),
		pix:  make([]byte, screenWidth*screenHeight*4),
	}
}

func IX(i, j int) int {
	return i + (N+2)*j
}

// --- シミュレーションコアロジック (Stamのアルゴリズム) ---

func (g *Game) diffuse(b int, x, x0 []float64, diff, dt float64) {
	a := dt * diff * float64(N) * float64(N)
	for k := 0; k < 20; k++ {
		for i := 1; i <= N; i++ {
			for j := 1; j <= N; j++ {
				x[IX(i, j)] = (x0[IX(i, j)] + a*(x[IX(i-1, j)]+x[IX(i+1, j)]+x[IX(i, j-1)]+x[IX(i, j+1)])) / (1 + 4*a)
			}
		}
		g.setBoundary(b, x)
	}
}

func (g *Game) advect(b int, d, d0, u, v []float64, dt float64) {
	dt0 := dt * float64(N)
	for i := 1; i <= N; i++ {
		for j := 1; j <= N; j++ {
			x := float64(i) - dt0*u[IX(i, j)]
			y := float64(j) - dt0*v[IX(i, j)]

			if x < 0.5 { x = 0.5 }
			if x > float64(N)+0.5 { x = float64(N) + 0.5 }
			i0 := math.Floor(x)
			i1 := i0 + 1

			if y < 0.5 { y = 0.5 }
			if y > float64(N)+0.5 { y = float64(N) + 0.5 }
			j0 := math.Floor(y)
			j1 := j0 + 1

			s1 := x - i0
			s0 := 1.0 - s1
			t1 := y - j0
			t0 := 1.0 - t1

			d[IX(i, j)] = s0*(t0*d0[IX(int(i0), int(j0))]+t1*d0[IX(int(i0), int(j1))]) +
				s1*(t0*d0[IX(int(i1), int(j0))]+t1*d0[IX(int(i1), int(j1))])
		}
	}
	g.setBoundary(b, d)
}

// 速度場を「質量保存（圧縮のない流体）」に強制する処理。これがないと渦が巻かない。
func (g *Game) project(u, v, p, div []float64) {
	for i := 1; i <= N; i++ {
		for j := 1; j <= N; j++ {
			div[IX(i, j)] = -0.5 * (u[IX(i+1, j)] - u[IX(i-1, j)] + v[IX(i, j+1)] - v[IX(i, j-1)]) / float64(N)
			p[IX(i, j)] = 0
		}
	}
	g.setBoundary(0, div)
	g.setBoundary(0, p)

	for k := 0; k < 20; k++ {
		for i := 1; i <= N; i++ {
			for j := 1; j <= N; j++ {
				p[IX(i, j)] = (div[IX(i, j)] + p[IX(i-1, j)] + p[IX(i+1, j)] + p[IX(i, j-1)] + p[IX(i, j+1)]) / 4
			}
		}
		g.setBoundary(0, p)
	}

	for i := 1; i <= N; i++ {
		for j := 1; j <= N; j++ {
			u[IX(i, j)] -= 0.5 * float64(N) * (p[IX(i+1, j)] - p[IX(i-1, j)])
			v[IX(i, j)] -= 0.5 * float64(N) * (p[IX(i, j+1)] - p[IX(i, j-1)])
		}
	}
	g.setBoundary(1, u)
	g.setBoundary(2, v)
}

func (g *Game) setBoundary(b int, x []float64) {
	for i := 1; i <= N; i++ {
		if b == 1 { x[IX(0, i)] = -x[IX(1, i)] } else { x[IX(0, i)] = x[IX(1, i)] }
		if b == 1 { x[IX(N+1, i)] = -x[IX(N, i)] } else { x[IX(N+1, i)] = x[IX(N, i)] }
		if b == 2 { x[IX(i, 0)] = -x[IX(i, 1)] } else { x[IX(i, 0)] = x[IX(i, 1)] }
		if b == 2 { x[IX(i, N+1)] = -x[IX(i, N)] } else { x[IX(i, N+1)] = x[IX(i, N)] }
	}
	x[IX(0, 0)] = 0.5 * (x[IX(1, 0)] + x[IX(0, 1)])
	x[IX(0, N+1)] = 0.5 * (x[IX(1, N+1)] + x[IX(0, N)])
	x[IX(N+1, 0)] = 0.5 * (x[IX(N, 0)] + x[IX(N+1, 1)])
	x[IX(N+1, N+1)] = 0.5 * (x[IX(N, N+1)] + x[IX(N+1, N)])
}

// --- Ebitengine インタフェースの実装 ---

// Update は毎フレーム（60fps）呼ばれるロジック更新処理
func (g *Game) Update() error {
	dt := 0.1
	diff := 0.0001 // 密度の拡散率
	visc := 0.0001 // 粘性（速度の拡散率）

	// マウス入力を流体のソース（発生源）にする
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		mx, my := ebiten.CursorPosition()
		// 画面座標から格子インデックスに変換
		i := mx / scale
		j := my / scale

		if i > 0 && i <= N && j > 0 && j <= N {
			// クリックした場所に強い密度（煙）を注入
			g.d[IX(i, j)] = 15.0
			g.d[IX(i+1, j)] = 15.0
			g.d[IX(i, j+1)] = 15.0

			// マウスの動きに合わせて外力を与える（簡易的にランダムな渦を生成）
			g.u[IX(i, j)] = 5.0
			g.v[IX(i, j)] = -5.0
		}
	}

	// 1. 速度場の更新 (外力 -> 拡散 -> 質量保存 -> 移流 -> 質量保存)
	g.u, g.uOld = g.uOld, g.u
	g.diffuse(1, g.u, g.uOld, visc, dt)
	g.v, g.vOld = g.vOld, g.v
	g.diffuse(2, g.v, g.vOld, visc, dt)
	g.project(g.u, g.v, g.uOld, g.vOld)

	g.u, g.uOld = g.uOld, g.u
	g.v, g.vOld = g.vOld, g.v
	g.advect(1, g.u, g.uOld, g.uOld, g.vOld, dt)
	g.advect(2, g.v, g.vOld, g.uOld, g.vOld, dt)
	g.project(g.u, g.v, g.uOld, g.vOld)

	// 2. 密度（煙）の更新 (拡散 -> 移流)
	g.d, g.dOld = g.dOld, g.d
	g.diffuse(0, g.d, g.dOld, diff, dt)
	g.d, g.dOld = g.dOld, g.d
	g.advect(0, g.d, g.dOld, g.u, g.v, dt)

	// 煙をじわじわ自然消滅させる（減衰）
	for i := 0; i < size; i++ {
		g.d[i] *= 0.995
	}

	return nil
}

// Draw は画面のレンダリング（描画）処理
func (g *Game) Draw(screen *ebiten.Image) {
	// 各格子の密度を読み取り、対応するピクセル（RGBA）を書き換える
	for j := 0; j < N; j++ {
		for i := 0; i < N; i++ {
			// シミュレーション内のインデックス（境界+1のズレを考慮）
			simI := i + 1
			simJ := j + 1
			density := g.d[IX(simI, simJ)]

			// 密度に応じて煙の輝度を決定 (上限 255)
			val := uint8(math.Min(density*40, 255))

			// この格子が担当する画面上の『scale x scale』のピクセル領域を塗りつぶす
			for py := 0; py < scale; py++ {
				for px := 0; px < scale; px++ {
					screenX := i*scale + px
					screenY := j*scale + py
					idx := (screenY*screenWidth + screenX) * 4

					if idx >= 0 && idx < len(g.pix)-3 {
						// 美しいサイバーブルーの煙を表現
						g.pix[idx] = val/4   // R
						g.pix[idx+1] = val   // G
						g.pix[idx+2] = val   // B
						g.pix[idx+3] = 255   // A (不透明度)
					}
				}
			}
		}
	}

	// 作成したピクセルバッファを画面に一括転送
	screen.WritePixels(g.pix)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("2D Fluid Dynamics Simulation")
	if err := ebiten.RunGame(NewGame()); err != nil {
		log.Fatal(err)
	}
}