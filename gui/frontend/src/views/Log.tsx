import {Trash2} from "lucide-react";
import {ScrollArea, ScrollBar} from "@/components/ui/scroll-area.tsx";
import SyntaxHighlighter from "react-syntax-highlighter/dist/esm/light";
import {atomOneDark, atomOneLight} from "react-syntax-highlighter/dist/esm/styles/hljs"
import {ComponentProps, useCallback, useEffect, useMemo, useRef, useState} from "react";
import {useTheme} from "@/components/theme-provider";
import {EventsOn} from "../../wailsjs/runtime";
import {ConnectStatus} from "@/views/types.ts";
import {cn} from "@/lib/utils.ts";
import cliLog from '@/lib/golog.js';

const MAX_LOG_CHARS = 200_000;
const LOG_FLUSH_INTERVAL_MS = 100;

SyntaxHighlighter.registerLanguage('accesslog', cliLog);

interface LogViewProps {
  status: ConnectStatus
}

export default function LogView({status, className, ...props}: ComponentProps<'div'> & LogViewProps) {
  const [logValue, setLogValue] = useState<string>('');
  const pendingLogs = useRef<string[]>([]);
  const flushTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const {theme} = useTheme()

  // 改一下默认的 bg 样式
  const highlighterStyle = useMemo(() => {
    const baseTheme = theme === 'light' ? atomOneLight : atomOneDark;
    const mainStyleKey = 'hljs';
    return {
      ...baseTheme,
      [mainStyleKey]: {
        ...(baseTheme[mainStyleKey] || {}),
        background: 'inherit', // 或者 'transparent'
        padding: 0,
      },
    };
  }, [theme]);

  // logValue 改变时，自动滚动到底部
  const scrollEndRef = useRef(null);
  useEffect(() => {
    if (scrollEndRef.current) {
      scrollEndRef.current.scrollIntoView({behavior: 'auto', block: 'end'});
    }
  }, [logValue]);


  const flushLogs = useCallback(() => {
    flushTimer.current = null;
    if (pendingLogs.current.length === 0) return;

    const incoming = pendingLogs.current.join('');
    pendingLogs.current = [];
    setLogValue((previous) => {
      const next = previous + incoming;
      if (next.length <= MAX_LOG_CHARS) return next;

      const tail = next.slice(-MAX_LOG_CHARS);
      const firstLineEnd = tail.indexOf('\n');
      return firstLineEnd === -1 ? tail : tail.slice(firstLineEnd + 1);
    });
  }, []);

  const onLogValue = useCallback((entry: string) => {
    pendingLogs.current.push(entry);
    if (flushTimer.current === null) {
      flushTimer.current = setTimeout(flushLogs, LOG_FLUSH_INTERVAL_MS);
    }
  }, [flushLogs]);

  const clearLogs = useCallback(() => {
    pendingLogs.current = [];
    if (flushTimer.current !== null) {
      clearTimeout(flushTimer.current);
      flushTimer.current = null;
    }
    setLogValue('');
  }, []);

  useEffect(() => {
    const unsubscribe = EventsOn('log', onLogValue)
    return () => {
      unsubscribe()
      if (flushTimer.current !== null) {
        clearTimeout(flushTimer.current)
      }
    }
  }, [onLogValue]);

  useEffect(() => {
    if (status === ConnectStatus.CONNECTING) {
      clearLogs()
    }
  }, [status, clearLogs]);

  const baseClass = "flex flex-1 flex-col space-y-4 bg-muted border border-border/50 rounded [&>p]:m-0 [&>p]:text-sm min-h-0"

  return (
    <div
      className={cn(baseClass, className)} {...props}>
      <div className="flex justify-between items-center shrink-0 px-4 pt-4">
        <p className="text-sm">运行日志</p>
        <Trash2 size={18} strokeWidth={1.6} onClick={clearLogs}/>
      </div>
      <ScrollArea className="flex flex-1 min-h-0 text-muted-foreground text-sm px-4 pb-4">
        <SyntaxHighlighter
          children={logValue}
          style={highlighterStyle}
          useInlineStyles={true}
          language="accesslog"
        />
        <div ref={scrollEndRef}/>
        <ScrollBar orientation="horizontal"/>
      </ScrollArea>
    </div>
  )
}
