import { Container } from '@cloudflare/containers'
import { env } from 'cloudflare:workers'

// The Go backend container
export class ImmanentTech extends Container {
  envVars = {
    IMMANENT_TECH_WEB_ENVIRONMENT: env.IMMANENT_TECH_WEB_ENVIRONMENT,
    IMMANENT_TECH_WEB_LOGLEVEL: env.IMMANENT_TECH_WEB_LOGLEVEL,
    IMMANENT_TECH_WEB_HOST: env.IMMANENT_TECH_WEB_HOST,
    IMMANENT_TECH_WEB_PORT: env.IMMANENT_TECH_WEB_PORT,
    IMMANENT_TECH_WEB_READTIMEOUT: env.IMMANENT_TECH_WEB_READTIMEOUT,
    IMMANENT_TECH_WEB_WRITETIMEOUT: env.IMMANENT_TECH_WEB_WRITETIMEOUT,
    IMMANENT_TECH_WEB_IDLETIMEOUT: env.IMMANENT_TECH_WEB_IDLETIMEOUT,
    IMMANENT_TECH_WEB_BASEURL: env.IMMANENT_TECH_WEB_BASEURL,
    IMMANENT_TECH_WEB_CSP_BASEURI: env.IMMANENT_TECH_WEB_CSP_BASEURI,
    IMMANENT_TECH_WEB_CSP_DEFAULTSRC: env.IMMANENT_TECH_WEB_CSP_DEFAULTSRC,
    IMMANENT_TECH_WEB_CSP_STYLESRC: env.IMMANENT_TECH_WEB_CSP_STYLESRC,
    IMMANENT_TECH_WEB_CSP_SCRIPTSRC: env.IMMANENT_TECH_WEB_CSP_SCRIPTSRC,
    IMMANENT_TECH_WEB_CSP_IMGSRC: env.IMMANENT_TECH_WEB_CSP_IMGSRC,
    IMMANENT_TECH_WEB_CSP_MEDIASRC: env.IMMANENT_TECH_WEB_CSP_MEDIASRC,
    IMMANENT_TECH_WEB_CSP_FONTSRC: env.IMMANENT_TECH_WEB_CSP_FONTSRC,
    IMMANENT_TECH_WEB_CSP_CONNECTSRC: env.IMMANENT_TECH_WEB_CSP_CONNECTSRC,
    IMMANENT_TECH_WEB_CORS_ALLOWEDORIGINS:
      env.IMMANENT_TECH_WEB_CORS_ALLOWEDORIGINS,
    IMMANENT_TECH_WEB_CORS_MAXAGE: env.IMMANENT_TECH_WEB_CORS_MAXAGE,
    IMMANENT_TECH_WEB_UMAMI_ID: env.IMMANENT_TECH_WEB_UMAMI_ID,
  }

  defaultPort = env.IMMANENT_TECH_WEB_PORT // Port your Go app listens on

  override sleepAfter = '2m' // Keep warm for 10 minutes after last request

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
      const id = env.GO_BACKEND.idFromName('www')
      const container = env.GO_BACKEND.get(id)

      // Forward the request as-is to the container
      return await container.fetch(request)
    } catch (err) {
      console.error('Failed to route to backend:', err)
      return new Response('Backend unavailable', { status: 502 })
    }
  },
} satisfies ExportedHandler<Env>
