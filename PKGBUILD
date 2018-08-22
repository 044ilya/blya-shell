pkgname=blya-shell
pkgver=1.0
pkgrel=0
pkgdesc="Bilyk Ilya Shell written in Go"
arch=('x86_64' 'armv7h' 'aarch64')
url="https://github.com/dwarq7/$pkgname"
license=('MIT')
makedepends=('git' 'go')
backup=()
options=("!strip")
source=("git://github.com/dwarq7/$pkgname.git")
sha256sums=('SKIP')

prepare() {
	msg2 "Download dependencies"
	
	export GOPATH="$startdir"
	
	cd "$srcdir/$pkgname"
	make deps
}

build() {
	cd "$srcdir/$pkgname"
	make build
}

package() {
	cd "$srcdir/$pkgname"
	
	export DESTDIR="$pkgdir"
	make install
	make clean
}

