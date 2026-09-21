// Package update は確認済みのリリースに添付された install.sh を取得して実行する。
// 検証も置換もインストーラーが持つので Go 側には持たない。取得と実行の実装は
// noupdate タグのビルドから外れる。方針は docs/design/updates.md にある。
package update
