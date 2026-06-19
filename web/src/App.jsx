import React, { useState } from 'react'
import InterpreterSelect from './components/InterpreterSelect'

export default function App(){
  const interpreters = [
    {id:'node-14', label:'Node.js 14.x'},
    {id:'python-3.8', label:'Python 3.8'},
    {id:'python-3.9', label:'Python 3.9'},
  ]
  const [selected, setSelected] = useState('')
  const [email, setEmail] = useState('')
  const [discord, setDiscord] = useState('')
  const [facebook, setFacebook] = useState('')
  const [status, setStatus] = useState(null)

  async function submit(e){
    e.preventDefault()
    setStatus('sending')
    const payload = { interpreter: selected, email, discord, facebook }
    try{
      const res = await fetch('/api/subscribe', { method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify(payload) })
      const j = await res.json()
      if(res.ok) setStatus('ok')
      else setStatus('error:'+ (j.message||res.status))
    }catch(err){
      setStatus('error:'+err.message)
    }
  }

  return (
    <div className="container">
      <h1 className="title">Deprecation Notifier</h1>
      <form onSubmit={submit} className="card">
        <InterpreterSelect items={interpreters} value={selected} onChange={setSelected} />

        <label className="field">
          <div className="label">Email</div>
          <input type="email" value={email} onChange={e=>setEmail(e.target.value)} placeholder="you@example.com" />
        </label>

        <label className="field">
          <div className="label">Discord (numeric ID)</div>
          <input value={discord} onChange={e=>setDiscord(e.target.value)} placeholder="Discord numeric ID (see help)" />
          <div className="help">How to get your Discord ID: enable Developer Mode in User Settings → Advanced. Then right-click the user in Desktop and choose "Copy ID". On mobile enable Developer Mode and long‑press the profile to copy ID.</div>
        </label>

        <label className="field">
          <div className="label">Facebook (experimental)</div>
          <input value={facebook} onChange={e=>setFacebook(e.target.value)} placeholder="Facebook ID / PSID (experimental)" />
          <div className="help">Facebook delivery is experimental: you may need to first message the page so we can capture your PSID. See the documentation for details.</div>
        </label>

        <div className="muted">Choose exactly one interpreter to subscribe to.</div>

        <button className="btn" type="submit" disabled={!selected}>Subscribe</button>
        {status && <div className="status">{status}</div>}
      </form>

      <footer className="muted">Note: Discord delivery requires a bot and may need you to share server access. We only collect IDs here.</footer>
    </div>
  )
}
