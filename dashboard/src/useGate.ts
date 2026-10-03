import { useEffect, useRef, useState } from 'react'
import { api, type Agent, type Config, type GateEvent, type Review } from './api'

const MAX_EVENTS = 2000

// Polls the gate: events incrementally every second, agents every two.
export function useGate() {
  const [events, setEvents] = useState<GateEvent[]>([])
  const [agents, setAgents] = useState<Agent[]>([])
  const [config, setConfig] = useState<Config | null>(null)
  const [reviews, setReviews] = useState<Review[]>([])
  const [error, setError] = useState<string | null>(null)
  const lastId = useRef(0)

  useEffect(() => {
    let alive = true
    let busy = false
    const pollEvents = async () => {
      if (busy) return
      busy = true
      try {
        const fresh = await api.events(lastId.current)
        if (alive) setError(null)
        if (!alive || fresh.length === 0) return
        lastId.current = fresh[fresh.length - 1].id
        setEvents((prev) => [...prev, ...fresh].slice(-MAX_EVENTS))
      } catch (e) {
        if (alive) setError(String(e))
      } finally {
        busy = false
      }
    }
    const pollAgents = async () => {
      try {
        const [a, r] = await Promise.all([api.agents(), api.reviews()])
        if (!alive) return
        setAgents(a)
        setReviews(r)
      } catch (e) {
        if (alive) setError(String(e))
      }
    }
    api.config().then((c) => alive && setConfig(c)).catch((e) => alive && setError(String(e)))
    pollEvents()
    pollAgents()
    const t1 = setInterval(pollEvents, 1000)
    const t2 = setInterval(pollAgents, 2000)
    return () => {
      alive = false
      clearInterval(t1)
      clearInterval(t2)
    }
  }, [])

  const refreshAgents = () => api.agents().then(setAgents)
  // After a review decision the vault and policy change, so reload both.
  const refreshReviews = () => Promise.all([api.reviews().then(setReviews), api.config().then(setConfig)])
  return { events, agents, config, reviews, error, refreshAgents, refreshReviews }
}
