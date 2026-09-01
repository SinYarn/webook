import React, {useEffect, useState} from "react";
import {Modal, Spin} from "antd";
import axios from "@/axios/axios";
import {canPreview, extension, fileKind} from "./FileThumb";

function apiBase(): string {
    return String(axios.defaults.baseURL || "").replace(/\/$/, "");
}

async function issueRawURL(id: number): Promise<string> {
    const res = await axios.post("/files/preview-ticket", {id});
    if (res.status != 200 || !res.data || !res.data.ticket) {
        throw new Error(typeof res.data === "string" ? res.data : "无法预览");
    }
    return apiBase() + "/files/raw?id=" + id + "&ticket=" + encodeURIComponent(res.data.ticket);
}

export function FilePreview({row, onClose}: {row: UserFile | null; onClose: () => void}) {
    const [url, setUrl] = useState("");
    const [text, setText] = useState("");
    const [textReady, setTextReady] = useState(false);
    const [err, setErr] = useState("");
    const [loading, setLoading] = useState(false);

    useEffect(() => {
        setUrl("");
        setText("");
        setTextReady(false);
        setErr("");
        if (!row || !canPreview(row)) {
            return;
        }
        let cancelled = false;
        setLoading(true);
        issueRawURL(row.Id).then(async (u) => {
            if (cancelled) {
                return;
            }
            const kind = fileKind(row);
            const ext = extension(row.Filename);
            if (kind == "text" || ext == "md" || ext == "txt") {
                const r = await fetch(u);
                if (!r.ok) {
                    throw new Error("读取失败");
                }
                const body = await r.text();
                if (!cancelled) {
                    setText(body);
                    setTextReady(true);
                }
                return;
            }
            setUrl(u);
        }).catch((e) => {
            if (!cancelled) {
                setErr(String(e?.message || e));
            }
        }).finally(() => {
            if (!cancelled) {
                setLoading(false);
            }
        });
        return () => {
            cancelled = true;
        };
    }, [row]);

    if (!row) {
        return null;
    }
    const kind = fileKind(row);
    const ext = extension(row.Filename);

    let body: React.ReactNode = null;
    if (loading) {
        body = <div style={{textAlign: "center", padding: 48}}><Spin/></div>;
    } else if (err) {
        body = <div style={{color: "#ff4d4f"}}>{err}</div>;
    } else if (kind == "image" && url) {
        body = <img alt={row.Filename} src={url} style={{maxWidth: "100%", maxHeight: "70vh", display: "block", margin: "0 auto"}}/>;
    } else if (kind == "video" && url) {
        body = <>
            <video key={url} controls autoPlay src={url} style={{width: "100%", maxHeight: "70vh", background: "#000"}}/>
            <div style={{marginTop: 8, color: "#8c8c8c", fontSize: 12}}>拖进度条走 Range。若无法播放（如部分 MOV），请下载到本地。</div>
        </>;
    } else if (kind == "audio" && url) {
        body = <>
            <audio key={url} controls autoPlay src={url} style={{width: "100%", marginTop: 24}}/>
            <div style={{marginTop: 8, color: "#8c8c8c", fontSize: 12}}>拖进度条走 Range。FLAC 等格式若无法播放，请下载到本地。</div>
        </>;
    } else if (kind == "pdf" && url) {
        body = <iframe title={row.Filename} src={url} style={{width: "100%", height: "70vh", border: 0}}/>;
    } else if ((kind == "text" || ext == "md" || ext == "txt") && textReady) {
        body = <pre style={{
            whiteSpace: "pre-wrap",
            wordBreak: "break-word",
            maxHeight: "70vh",
            overflow: "auto",
            margin: 0,
            fontSize: 13,
            lineHeight: 1.6,
        }}>{text}</pre>;
    } else if (!loading && !err) {
        body = <div>无法预览</div>;
    }

    return (
        <Modal
            title={row.Filename}
            open={true}
            onCancel={onClose}
            footer={null}
            width={kind == "pdf" || kind == "video" ? 900 : 720}
            destroyOnClose
        >
            {body}
        </Modal>
    );
}

export default function FilePreviewPage() {
    return null;
}
