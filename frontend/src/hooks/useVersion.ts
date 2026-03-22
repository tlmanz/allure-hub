import { useEffect, useState } from 'react'

export interface VersionInfo {
  version: string
  buildTime: string
  goVersion: string
}

export function useVersion(): VersionInfo | null {
  const [info, setInfo] = useState<VersionInfo | null>(null)

  useEffect(() => {
    fetch('/api/version')
      .then(r => r.json())
      .then((data: unknown) => {
        if (
          data !== null &&
          typeof data === 'object' &&
          typeof (data as Record<string, unknown>).version === 'string' &&
          typeof (data as Record<string, unknown>).buildTime === 'string' &&
          typeof (data as Record<string, unknown>).goVersion === 'string'
        ) {
          setInfo(data as VersionInfo)
        }
      })
      .catch(() => {})
  }, [])

  return info
}
