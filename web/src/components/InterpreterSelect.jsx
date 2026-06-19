import React from 'react'

export default function InterpreterSelect({items,value,onChange}){
  return (
    <div className="field">
      <div className="label">Select one interpreter</div>
      <div className="grid">
        {items.map(it=> (
          <label key={it.id} className={`chip ${value===it.id?'selected':''}`} onClick={()=>onChange(it.id)}>
            <input type="radio" name="interpreter" checked={value===it.id} onChange={()=>onChange(it.id)} />
            <span>{it.label}</span>
          </label>
        ))}
      </div>
    </div>
  )
}
