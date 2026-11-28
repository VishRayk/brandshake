import QRCode from 'qrcode'
import { useEffect, useState } from 'react'

/** QR code for a upi://pay link; any UPI app can scan it. */
export function UpiQr({ uri, size = 168 }: { uri: string; size?: number }) {
  const [src, setSrc] = useState<string>()
  useEffect(() => {
    let alive = true
    QRCode.toDataURL(uri, {
      margin: 1,
      width: size * 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#1c1917', light: '#ffffff' },
    })
      .then((url) => alive && setSrc(url))
      .catch(() => alive && setSrc(undefined))
    return () => {
      alive = false
    }
  }, [uri, size])
  if (!src)
    return (
      <div
        className="animate-pulse rounded-md bg-neutral-subtle"
        style={{ width: size, height: size }}
      />
    )
  return (
    <img
      src={src}
      width={size}
      height={size}
      alt="UPI payment QR code"
      className="rounded-md border border-line bg-white p-1"
    />
  )
}
