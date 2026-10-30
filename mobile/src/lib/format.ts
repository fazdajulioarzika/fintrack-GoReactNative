// Format rupiah manual (tidak bergantung pada Intl yang dukungannya beda-beda di tiap perangkat)
export function formatIDR(amount: number): string {
  const sign = amount < 0 ? "-" : "";
  const digits = Math.abs(Math.trunc(amount))
    .toString()
    .replace(/\B(?=(\d{3})+(?!\d))/g, ".");
  return `${sign}Rp${digits}`;
}
