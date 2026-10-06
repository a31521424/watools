import {useEffect, type ReactNode} from 'react'
import {ArrowLeft} from 'lucide-react'
import {useLocation} from 'wouter'
import {Button} from '@/components/ui/button'

export function WaWorkspaceHeader({title, subtitle, icon, actions}: {
    title: string; subtitle?: string; icon: ReactNode; actions?: ReactNode
}) {
    const [, navigate] = useLocation()
    useEffect(() => {
        const returnToLauncher = (event: KeyboardEvent) => {
            if (event.defaultPrevented || event.isComposing || event.keyCode === 229 || event.key !== 'Escape') return
            event.preventDefault()
            navigate('/')
        }
        window.addEventListener('keydown', returnToLauncher)
        return () => window.removeEventListener('keydown', returnToLauncher)
    }, [navigate])
    return <header className="wa-workspace-header">
        <Button variant="ghost" size="sm" className="wa-back-button" onClick={() => navigate('/')} aria-label="返回启动器" title="返回启动器 · Esc">
            <ArrowLeft size={16}/>
        </Button>
        <div className="wa-workspace-icon" aria-hidden="true">{icon}</div>
        <div className="wa-workspace-identity">
            <div className="wa-workspace-title"><h1>{title}</h1><span>WaTools</span></div>
            {subtitle && <p title={subtitle}>{subtitle}</p>}
        </div>
        {actions && <div className="wa-workspace-actions">{actions}</div>}
    </header>
}
