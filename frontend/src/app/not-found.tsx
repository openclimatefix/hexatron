import Link from 'next/link'

export default function NotFound() {
  return (
    <div className="flex flex-1 items-center justify-center bg-canvas px-6 py-20">
      <div className="max-w-md text-center">
        <h1 className="font-display text-2xl font-medium tracking-tight text-ink">Not found</h1>
        <p className="mt-2 text-sm text-black/55">That service isn’t in the registry.</p>
        <Link
          href="/"
          className="mt-6 inline-block text-sm text-black/50 underline-offset-4 transition-colors hover:text-black/75 hover:underline"
        >
          <span aria-hidden>←</span> Back to dashboard
        </Link>
      </div>
    </div>
  )
}
