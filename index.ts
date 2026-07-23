import { Container } from '@cloudflare/containers'
import { env } from 'cloudflare:workers'

// The Go backend container
export class ImmanentTech extends Container {
  envVars = {
    LOG_LEVEL: env.LOG_LEVEL,
    WWW_PORT: env.WWW_PORT,
    APP_NAME: env.APP_NAME,
    APP_DESCRIPTION: env.APP_DESCRIPTION,
    APP_ID: env.APP_ID,
    APP_VERSION: env.APP_VERSION,
    APP_ENVIRONMENT: env.APP_ENVIRONMENT,
    APP_BASEURL: env.APP_BASEURL,
    CSP_SCRIPTSRC: env.CSP_SCRIPTSRC,
    CSP_FRAMESRC: env.CSP_FRAMESRC,
    CSP_CONNECTSRC: env.CSP_CONNECTSRC,
    CSP_IMGSRC: env.CSP_IMGSRC,
    CORS_ALLOWEDORIGINS: env.CORS_ALLOWEDORIGINS,
    CORS_MAXAGE: env.CORS_MAXAGE,
    UMAMI_ID: env.UMAMI_ID,
    CLOUDFLARE_TURNSTILE_KEY: env.CLOUDFLARE_TURNSTILE_KEY,
    FASTMAIL_APIKEY: env.FASTMAIL_APIKEY,
    FASTMAIL_IDENTITY: env.FASTMAIL_IDENTITY,
  }

  defaultPort = env.WWW_PORT // Port your Go app listens on

  override sleepAfter = '2m' // Keep warm for 2 minutes after last request

  override onStart() {
    console.log('Go backend container started')
  }

  override onStop() {
    console.log('Go backend container stopped')
  }

  override onError(error: unknown) {
    console.error('Go backend container error:', error)
  }
}

export default {
  async fetch(
    request: Request,
    env: Env,
    ctx: ExecutionContext
  ): Promise<Response> {
    try {
      // Get (or start) the singleton container instance
      const id = env.GO_BACKEND.idFromName('www-immanenttech')
      const container = env.GO_BACKEND.get(id)

      // Forward the request as-is to the container
      return await container.fetch(request)
    } catch (err) {
      console.error('Failed to route to backend:', err)
      return new Response('Backend unavailable', { status: 502 })
    }
  },
} satisfies ExportedHandler<Env>
