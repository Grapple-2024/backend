import { clerkMiddleware, createRouteMatcher } from '@clerk/nextjs/server'
import { NextResponse } from 'next/server'

const isPublicRoute = createRouteMatcher([
  '/',
  '/auth(.*)',
  '/profile(.*)',
  '/coach(.*)',
  '/student(.*)',
])

const isAuthSignInPage = createRouteMatcher(['/auth'])

export default clerkMiddleware((auth, request) => {
  const { userId } = auth()

  // Redirect already-signed-in users away from the sign-in page
  if (userId && isAuthSignInPage(request)) {
    return NextResponse.redirect(new URL('/auth/account-type', request.url))
  }

  // Protect non-public routes — redirect unauthenticated users to /auth
  if (!isPublicRoute(request) && !userId) {
    return NextResponse.redirect(new URL('/auth', request.url))
  }
})

export const config = {
  matcher: [
    '/((?!_next|[^?]*\\.(?:html?|css|js(?!on)|jpe?g|webp|png|gif|svg|ttf|woff2?|ico|csv|docx?|xlsx?|zip|webmanifest)).*)',
    '/(api|trpc)(.*)',
  ],
}
