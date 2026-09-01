import React, {useEffect, useRef, useState} from 'react';
import {Breadcrumb, Button, Input, Modal, Progress, Radio, Table, Upload} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import axios from "@/axios/axios";
import Link from "next/link";
import moment from "moment";
import SparkMD5 from "spark-md5";
import FileThumb, {canPreview} from "./FileThumb";
import {FilePreview} from "./FilePreview";

const CHUNK_SIZE = 2 * 1024 * 1024;

type TransferPhase = "hash" | "run" | "merge" | "paused" | "done" | "error";

type TransferJob = {
    kind: "upload" | "download";
    percent: number;
    phase: TransferPhase;
    message: string;
    mode?: "basic" | "chunk" | "folder";
};

type JobCtl = {
    abort?: AbortController;
    paused: boolean;
    canceled: boolean;
    file?: File;
    dir?: number;
    identifier?: string;
    chunks?: Uint8Array[];
    received?: number;
    total?: number;
    row?: UserFile;
    reload?: boolean;
};

function fileSizeDesc(n: number): string {
    if (n < 1024) {
        return n + " B";
    }
    if (n < 1024 * 1024) {
        return (n / 1024).toFixed(1) + " KB";
    }
    if (n < 1024 * 1024 * 1024) {
        return (n / (1024 * 1024)).toFixed(1) + " MB";
    }
    return (n / (1024 * 1024 * 1024)).toFixed(1) + " GB";
}

function progressPercent(e: {loaded: number; total?: number}): number | null {
    const total = e.total || 0;
    if (total <= 0) {
        return null;
    }
    return Math.min(100, Math.round(e.loaded / total * 100));
}

function isAbort(err: any): boolean {
    return err?.code === "ERR_CANCELED" || err?.name === "AbortError";
}

function apiBase(): string {
    return String(axios.defaults.baseURL || "").replace(/\/$/, "");
}

function parseContentRange(h: string | null): {total: number} | null {
    const m = /^bytes \d+-\d+\/(\d+)$/.exec(h || "");
    if (!m) {
        return null;
    }
    return {total: Number(m[1])};
}

function relativePath(file: File): string {
    const p = ((file as File & {webkitRelativePath?: string}).webkitRelativePath || file.name).replace(/\\/g, "/");
    return p.replace(/^\/+/, "");
}

function skipJunk(name: string): boolean {
    return name === ".DS_Store" || name === "Thumbs.db" || name === "desktop.ini";
}

function saveBlob(blob: Blob, filename: string) {
    const url = window.URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
    window.URL.revokeObjectURL(url);
}

function md5File(file: File, onProgress?: (done: number, total: number) => void, stopped?: () => boolean): Promise<string> {
    return new Promise((resolve, reject) => {
        const chunks = Math.ceil(file.size / CHUNK_SIZE) || 1;
        const spark = new SparkMD5.ArrayBuffer();
        const reader = new FileReader();
        let i = 0;
        reader.onload = (e) => {
            if (stopped?.()) {
                reject(new DOMException("paused", "AbortError"));
                return;
            }
            spark.append(e.target!.result as ArrayBuffer);
            i++;
            onProgress?.(i, chunks);
            if (i < chunks) {
                loadNext();
            } else {
                resolve(spark.end());
            }
        };
        reader.onerror = reject;
        const loadNext = () => {
            if (stopped?.()) {
                reject(new DOMException("paused", "AbortError"));
                return;
            }
            const start = i * CHUNK_SIZE;
            const end = Math.min(start + CHUNK_SIZE, file.size);
            reader.readAsArrayBuffer(file.slice(start, end));
        };
        loadNext();
    });
}

function TransferCell({job}: {job?: TransferJob}) {
    if (!job) {
        return null;
    }
    const status = job.phase === "error"
        ? "exception"
        : job.phase === "done"
            ? "success"
            : job.phase === "paused"
                ? "normal"
                : "active";
    const color = job.kind === "download" ? "#52c41a" : undefined;
    return (
        <div style={{display: "flex", alignItems: "center", gap: 8, minWidth: 0}}>
            <Progress
                percent={job.percent < 0 ? 0 : job.percent}
                size="small"
                status={status as any}
                showInfo={false}
                strokeColor={job.phase === "error" || job.phase === "done" || job.phase === "paused" ? undefined : color}
                style={{flex: 1, margin: 0, minWidth: 72}}
            />
            <span style={{
                fontSize: 12,
                color: job.phase === "error" ? "#ff4d4f" : "#8c8c8c",
                whiteSpace: "nowrap",
            }}>
                {job.message}
            </span>
        </div>
    );
}

function Page() {
    const [parentId, setParentId] = useState(0);
    const [list, setList] = useState<UserFile[]>([]);
    const [pending, setPending] = useState<UserFile[]>([]);
    const [crumbs, setCrumbs] = useState<FileBreadcrumb[]>([]);
    const [loading, setLoading] = useState(false);
    const [mode, setMode] = useState("basic");
    const [folderOpen, setFolderOpen] = useState(false);
    const [folderName, setFolderName] = useState("");
    const [selected, setSelected] = useState<number[]>([]);
    const [jobs, setJobs] = useState<Record<string, TransferJob>>({});
    const [preview, setPreview] = useState<UserFile | null>(null);
    const parentIdRef = useRef(parentId);
    const timersRef = useRef<number[]>([]);
    const ctlRef = useRef<Record<string, JobCtl>>({});
    const folderInputRef = useRef<HTMLInputElement | null>(null);

    parentIdRef.current = parentId;

    useEffect(() => {
        return () => {
            timersRef.current.forEach((t) => window.clearTimeout(t));
        };
    }, []);

    const getCtl = (key: string): JobCtl => {
        if (!ctlRef.current[key]) {
            ctlRef.current[key] = {paused: false, canceled: false};
        }
        return ctlRef.current[key];
    };

    const patchJob = (key: string, patch: Partial<TransferJob>) => {
        setJobs((prev) => {
            const cur = prev[key] || {kind: "upload", percent: 0, phase: "run", message: ""};
            return {...prev, [key]: {...cur, ...patch}};
        });
    };

    const dropJob = (key: string) => {
        delete ctlRef.current[key];
        setJobs((prev) => {
            const next = {...prev};
            delete next[key];
            return next;
        });
    };

    const later = (fn: () => void, ms: number) => {
        const t = window.setTimeout(fn, ms);
        timersRef.current.push(t);
    };

    const load = (pid: number) => {
        setLoading(true);
        axios.get("/files", {params: {parentId: pid}})
            .then((res) => {
                if (res.status != 200) {
                    alert(res.statusText);
                    return;
                }
                if (!Array.isArray(res.data)) {
                    alert(res.data);
                    return;
                }
                setList(res.data);
            }).catch((err) => {
            alert(err);
        }).finally(() => {
            setLoading(false);
        });
        if (pid == 0) {
            setCrumbs([]);
            return;
        }
        axios.get("/files/breadcrumbs", {params: {id: pid}})
            .then((res) => {
                if (Array.isArray(res.data)) {
                    setCrumbs(res.data);
                }
            });
    };

    useEffect(() => {
        load(parentId);
    }, [parentId]);

    useEffect(() => {
        const el = folderInputRef.current;
        if (!el) {
            return;
        }
        el.setAttribute("webkitdirectory", "");
        el.setAttribute("directory", "");
    }, []);

    const finishUpload = (key: string, dir: number, ok: boolean, message: string) => {
        if (ok) {
            patchJob(key, {percent: 100, phase: "done", message: message || "完成"});
            later(() => {
                setPending((p) => p.filter((r) => r.RowKey !== key));
                const reload = getCtl(key).reload !== false;
                dropJob(key);
                if (reload && parentIdRef.current === dir) {
                    load(dir);
                }
            }, 2000);
            return;
        }
        patchJob(key, {phase: "error", message: message || "失败"});
    };

    const finishDownload = (key: string, ok: boolean, message: string) => {
        if (ok) {
            patchJob(key, {percent: 100, phase: "done", message: "完成"});
            later(() => dropJob(key), 2000);
            return;
        }
        patchJob(key, {phase: "error", message: message || "失败"});
    };

    const basicUpload = (file: File, key: string, dir: number) => {
        const ctl = getCtl(key);
        ctl.file = file;
        ctl.dir = dir;
        const ac = new AbortController();
        ctl.abort = ac;
        const fd = new FormData();
        fd.append("file", file);
        fd.append("parentId", String(dir));
        return axios.post("/files/upload", fd, {
            signal: ac.signal,
            onUploadProgress: (e) => {
                const p = progressPercent(e);
                patchJob(key, {
                    kind: "upload",
                    mode: "basic",
                    phase: "run",
                    percent: p == null ? 0 : p,
                    message: p == null ? "上传中" : "上传中 " + p + "%",
                });
            },
        }).then((res) => {
            if (ctl.canceled) {
                return;
            }
            if (res.status != 200) {
                finishUpload(key, dir, false, res.statusText);
                return;
            }
            if (res.data !== "上传成功") {
                finishUpload(key, dir, false, String(res.data));
                return;
            }
            finishUpload(key, dir, true, "完成");
        }).catch((err) => {
            if (ctl.canceled || isAbort(err)) {
                return;
            }
            finishUpload(key, dir, false, String(err));
        });
    };

    const chunkUpload = async (file: File, key: string, dir: number) => {
        const ctl = getCtl(key);
        ctl.file = file;
        ctl.dir = dir;
        ctl.paused = false;
        ctl.canceled = false;
        const stopped = () => ctl.paused || ctl.canceled;
        try {
            if (!ctl.identifier) {
                patchJob(key, {kind: "upload", mode: "chunk", phase: "hash", percent: 0, message: "校验中"});
                ctl.identifier = await md5File(file, (done, total) => {
                    const p = Math.round(done / total * 100);
                    patchJob(key, {
                        kind: "upload",
                        mode: "chunk",
                        phase: "hash",
                        percent: p,
                        message: "校验中 " + p + "%",
                    });
                }, stopped);
                if (stopped()) {
                    if (ctl.canceled) {
                        return;
                    }
                    patchJob(key, {phase: "paused", message: "已暂停"});
                    return;
                }
                const ac = new AbortController();
                ctl.abort = ac;
                const sec = await axios.post("/files/sec-upload", {
                    parentId: dir,
                    filename: file.name,
                    identifier: ctl.identifier,
                }, {signal: ac.signal});
                if (sec.status != 200) {
                    finishUpload(key, dir, false, sec.statusText);
                    return;
                }
                if (sec.data === "秒传成功") {
                    finishUpload(key, dir, true, "秒传成功");
                    return;
                }
                if (sec.data !== "文件不存在") {
                    finishUpload(key, dir, false, String(sec.data));
                    return;
                }
            }
            const totalChunks = Math.ceil(file.size / CHUNK_SIZE) || 1;
            const uploadedRes = await axios.get("/files/chunk-upload", {params: {identifier: ctl.identifier}});
            const uploaded: number[] = uploadedRes.data?.UploadedChunks || [];
            const doneSet = new Set(uploaded);
            const setChunkProgress = (finished: number, inner: number, phase: TransferPhase = "run") => {
                const p = Math.min(99, Math.round((finished + inner) / totalChunks * 100));
                patchJob(key, {
                    kind: "upload",
                    mode: "chunk",
                    phase,
                    percent: p,
                    message: "上传中 " + Math.min(finished, totalChunks) + "/" + totalChunks,
                });
            };
            setChunkProgress(doneSet.size, 0);
            for (let i = 1; i <= totalChunks; i++) {
                if (stopped()) {
                    if (ctl.canceled) {
                        return;
                    }
                    patchJob(key, {phase: "paused", message: "已暂停"});
                    return;
                }
                if (doneSet.has(i)) {
                    continue;
                }
                const start = (i - 1) * CHUNK_SIZE;
                const end = Math.min(start + CHUNK_SIZE, file.size);
                const blob = file.slice(start, end);
                const fd = new FormData();
                fd.append("file", blob, file.name);
                fd.append("filename", file.name);
                fd.append("identifier", ctl.identifier || "");
                fd.append("totalChunks", String(totalChunks));
                fd.append("chunkNumber", String(i));
                fd.append("totalSize", String(file.size));
                const ac = new AbortController();
                ctl.abort = ac;
                const r = await axios.post("/files/chunk-upload", fd, {
                    signal: ac.signal,
                    onUploadProgress: (e) => {
                        const inner = e.total ? e.loaded / e.total : 0;
                        setChunkProgress(doneSet.size, inner);
                    },
                });
                if (r.status != 200) {
                    finishUpload(key, dir, false, r.statusText);
                    return;
                }
                if (typeof r.data === "string") {
                    finishUpload(key, dir, false, r.data);
                    return;
                }
                doneSet.add(i);
                setChunkProgress(doneSet.size, 0);
            }
            if (stopped()) {
                if (ctl.canceled) {
                    return;
                }
                patchJob(key, {phase: "paused", message: "已暂停"});
                return;
            }
            patchJob(key, {kind: "upload", mode: "chunk", phase: "merge", percent: 99, message: "合并中"});
            const ac = new AbortController();
            ctl.abort = ac;
            const merge = await axios.post("/files/merge", {
                parentId: dir,
                filename: file.name,
                identifier: ctl.identifier,
                totalSize: file.size,
            }, {signal: ac.signal});
            if (merge.status != 200) {
                finishUpload(key, dir, false, merge.statusText);
                return;
            }
            if (merge.data !== "上传成功") {
                finishUpload(key, dir, false, String(merge.data));
                return;
            }
            finishUpload(key, dir, true, "完成");
        } catch (err) {
            if (ctl.canceled || isAbort(err)) {
                if (ctl.paused && !ctl.canceled) {
                    patchJob(key, {phase: "paused", message: "已暂停"});
                }
                return;
            }
            finishUpload(key, dir, false, String(err));
        }
    };

    const startUpload = (file: File, destDir?: number, reload = true) => {
        const dir = destDir == null ? parentId : destDir;
        const key = "up-" + Date.now() + "-" + Math.random().toString(36).slice(2, 8);
        const row: UserFile = {
            Id: 0,
            ParentId: dir,
            Filename: file.name,
            FolderFlag: 0,
            FileSizeDesc: fileSizeDesc(file.size),
            Utime: 0,
            Pending: true,
            RowKey: key,
            FileSize: file.size,
        };
        setPending((p) => [row, ...p]);
        getCtl(key).reload = reload;
        patchJob(key, {kind: "upload", mode: mode === "chunk" ? "chunk" : "basic", percent: 0, phase: "run", message: "上传中"});
        return mode === "chunk" ? chunkUpload(file, key, dir) : basicUpload(file, key, dir);
    };

    const pauseJob = (key: string) => {
        const ctl = getCtl(key);
        ctl.paused = true;
        ctl.abort?.abort();
        patchJob(key, {phase: "paused", message: "已暂停"});
    };

    const resumeUpload = (key: string) => {
        const ctl = getCtl(key);
        if (!ctl.file || ctl.dir == null) {
            return;
        }
        ctl.paused = false;
        ctl.canceled = false;
        patchJob(key, {phase: "run", message: "继续上传"});
        chunkUpload(ctl.file, key, ctl.dir);
    };

    const cancelUpload = (key: string) => {
        const ctl = getCtl(key);
        ctl.canceled = true;
        ctl.paused = false;
        ctl.abort?.abort();
        if (key.indexOf("up-folder-") === 0) {
            return;
        }
        setPending((p) => p.filter((r) => r.RowKey !== key));
        dropJob(key);
    };

    const busy = (key: string) => {
        const job = jobs[key];
        return job && job.phase !== "done" && job.phase !== "error";
    };

    const runDownload = async (row: UserFile) => {
        const key = "id-" + row.Id;
        const ctl = getCtl(key);
        ctl.row = row;
        ctl.paused = false;
        ctl.canceled = false;
        ctl.chunks = ctl.chunks || [];
        ctl.received = ctl.received || 0;
        patchJob(key, {kind: "download", phase: "run", message: "下载中", percent: ctl.total ? Math.round((ctl.received || 0) / ctl.total * 100) : 0});
        const token = localStorage.getItem("token") || "";
        try {
            while (!ctl.canceled && !ctl.paused) {
                const ac = new AbortController();
                ctl.abort = ac;
                const res = await fetch(apiBase() + "/files/download?id=" + row.Id, {
                    headers: {
                        Authorization: "Bearer " + token,
                        Range: "bytes=" + (ctl.received || 0) + "-",
                    },
                    credentials: "include",
                    signal: ac.signal,
                });
                const ctype = res.headers.get("content-type") || "";
                if (!res.ok && res.status !== 206) {
                    finishDownload(key, false, res.statusText);
                    return;
                }
                if (res.status === 200 && ctype.indexOf("text") >= 0) {
                    finishDownload(key, false, await res.text());
                    return;
                }
                const ranged = parseContentRange(res.headers.get("content-range"));
                if (ranged) {
                    ctl.total = ranged.total;
                }
                if (!res.body) {
                    finishDownload(key, false, "下载失败");
                    return;
                }
                const reader = res.body.getReader();
                while (!ctl.canceled && !ctl.paused) {
                    const {done, value} = await reader.read();
                    if (done) {
                        break;
                    }
                    ctl.chunks!.push(value);
                    ctl.received = (ctl.received || 0) + value.byteLength;
                    const total = ctl.total || 0;
                    const p = total ? Math.min(99, Math.round(ctl.received / total * 100)) : 0;
                    patchJob(key, {
                        kind: "download",
                        phase: "run",
                        percent: p,
                        message: total ? "下载中 " + p + "%" : "下载中",
                    });
                    if (total && ctl.received >= total) {
                        break;
                    }
                }
                try {
                    await reader.cancel();
                } catch {
                }
                if (ctl.paused) {
                    patchJob(key, {phase: "paused", message: "已暂停"});
                    return;
                }
                if (ctl.canceled) {
                    return;
                }
                const total = ctl.total || 0;
                if (!total || (ctl.received || 0) >= total) {
                    break;
                }
            }
            if (ctl.canceled) {
                return;
            }
            if (ctl.paused) {
                patchJob(key, {phase: "paused", message: "已暂停"});
                return;
            }
            const blob = new Blob(ctl.chunks || []);
            saveBlob(blob, row.Filename);
            ctl.chunks = [];
            ctl.received = 0;
            ctl.total = 0;
            finishDownload(key, true, "完成");
        } catch (err) {
            if (ctl.paused) {
                patchJob(key, {phase: "paused", message: "已暂停"});
                return;
            }
            if (ctl.canceled || isAbort(err)) {
                return;
            }
            finishDownload(key, false, String(err));
        }
    };

    const downloadFile = (row: UserFile) => {
        const key = "id-" + row.Id;
        if (busy(key) && jobs[key]?.phase !== "paused") {
            return;
        }
        runDownload(row);
    };

    const resumeDownload = (key: string) => {
        const ctl = getCtl(key);
        if (!ctl.row) {
            return;
        }
        runDownload(ctl.row);
    };

    const cancelDownload = (key: string) => {
        const ctl = getCtl(key);
        ctl.canceled = true;
        ctl.paused = false;
        ctl.abort?.abort();
        ctl.chunks = [];
        ctl.received = 0;
        ctl.total = 0;
        dropJob(key);
    };

    const removePending = (key: string) => {
        cancelUpload(key);
    };

    const deleteIds = (ids: number[]) => {
        if (ids.length == 0) {
            alert("请选择文件");
            return;
        }
        if (!window.confirm("确认删除？")) {
            return;
        }
        axios.post("/files/delete", {ids}).then((res) => {
            if (res.status != 200) {
                alert(res.statusText);
                return;
            }
            alert(res.data);
            setSelected([]);
            load(parentId);
        }).catch((err) => {
            alert(err);
        });
    };

    const createFolder = () => {
        axios.post("/files/folder", {parentId, folderName}).then((res) => {
            if (res.status != 200) {
                alert(res.statusText);
                return;
            }
            if (!res.data || !res.data.Id) {
                alert(res.data);
                return;
            }
            setFolderOpen(false);
            setFolderName("");
            load(parentId);
        }).catch((err) => {
            alert(err);
        });
    };

    const ensureFolder = async (parent: number, name: string): Promise<number> => {
        const res = await axios.post("/files/folder", {parentId: parent, folderName: name});
        if (res.data && res.data.Id) {
            return res.data.Id as number;
        }
        throw new Error(typeof res.data === "string" ? res.data : "创建文件夹失败");
    };

    const uploadFolder = async (picked: File[]) => {
        const files = picked.filter((f) => !skipJunk(f.name));
        if (files.length == 0) {
            return;
        }
        const rootDir = parentId;
        const cache: Record<string, number> = {"": rootDir};
        const rootName = relativePath(files[0]).split("/")[0] || "文件夹";
        const batchKey = "up-folder-" + Date.now();
        const batchRow: UserFile = {
            Id: 0,
            ParentId: rootDir,
            Filename: rootName,
            FolderFlag: 1,
            FileSizeDesc: files.length + " 个文件",
            Utime: 0,
            Pending: true,
            RowKey: batchKey,
        };
        setPending((p) => [batchRow, ...p]);
        getCtl(batchKey).reload = false;
        patchJob(batchKey, {kind: "upload", mode: "folder", percent: 0, phase: "run", message: "0/" + files.length});
        const ensurePath = async (dirPath: string): Promise<number> => {
            if (cache[dirPath] != null) {
                return cache[dirPath];
            }
            const parts = dirPath.split("/").filter(Boolean);
            let pid = rootDir;
            let acc = "";
            for (const part of parts) {
                acc = acc ? acc + "/" + part : part;
                if (cache[acc] != null) {
                    pid = cache[acc];
                    continue;
                }
                pid = await ensureFolder(pid, part);
                cache[acc] = pid;
            }
            return pid;
        };
        try {
            for (let i = 0; i < files.length; i++) {
                if (getCtl(batchKey).canceled) {
                    break;
                }
                const file = files[i];
                const rel = relativePath(file);
                const slash = rel.lastIndexOf("/");
                const dirPath = slash < 0 ? "" : rel.slice(0, slash);
                const dest = await ensurePath(dirPath);
                const p = Math.round(i / files.length * 100);
                patchJob(batchKey, {
                    kind: "upload",
                    mode: "folder",
                    phase: "run",
                    percent: p,
                    message: (i + 1) + "/" + files.length,
                });
                await startUpload(file, dest, false);
            }
            if (getCtl(batchKey).canceled) {
                setPending((p) => p.filter((r) => r.RowKey !== batchKey));
                dropJob(batchKey);
                return;
            }
            patchJob(batchKey, {percent: 100, phase: "done", message: files.length + "/" + files.length});
            later(() => {
                setPending((p) => p.filter((r) => r.RowKey !== batchKey));
                dropJob(batchKey);
                if (parentIdRef.current === rootDir) {
                    load(rootDir);
                }
            }, 2000);
        } catch (err) {
            if (getCtl(batchKey).canceled || isAbort(err)) {
                return;
            }
            patchJob(batchKey, {phase: "error", message: String(err)});
        }
    };

    const rows: UserFile[] = [
        ...pending.filter((p) => p.ParentId === parentId),
        ...list.map((f) => ({...f, RowKey: "id-" + f.Id})),
    ];

    const transferButtons = (key: string, job?: TransferJob) => {
        if (!job || job.phase === "done" || job.phase === "error") {
            return null;
        }
        const canPause = job.phase !== "paused" && job.phase !== "merge" &&
            (job.kind === "download" || job.mode === "chunk");
        return <>
            {canPause && <Button type="link" onClick={() => pauseJob(key)}>暂停</Button>}
            {job.phase === "paused" && job.kind === "upload" &&
                <Button type="link" onClick={() => resumeUpload(key)}>继续</Button>}
            {job.phase === "paused" && job.kind === "download" &&
                <Button type="link" onClick={() => resumeDownload(key)}>继续</Button>}
            <Button type="link" danger onClick={() => job.kind === "download" ? cancelDownload(key) : cancelUpload(key)}>
                取消
            </Button>
        </>;
    };

    const columns: ColumnsType<UserFile> = [
        {
            title: "文件名",
            dataIndex: "Filename",
            render: (_, row) => {
                const name = row.FolderFlag == 1
                    ? <a onClick={() => setParentId(row.Id)}>{row.Filename}</a>
                    : canPreview(row)
                        ? <a onClick={() => setPreview(row)}>{row.Filename}</a>
                        : <span>{row.Filename}</span>;
                return <div style={{display: "flex", alignItems: "center", gap: 8}}>
                    <FileThumb row={row}/>
                    {name}
                </div>;
            },
        },
        {
            title: "大小",
            dataIndex: "FileSizeDesc",
            width: 120,
            render: (v) => v || "-",
        },
        {
            title: "修改日期",
            dataIndex: "Utime",
            width: 180,
            render: (v) => v ? moment(v).format("YYYY-MM-DD HH:mm:ss") : "-",
        },
        {
            title: "进度",
            width: 200,
            render: (_, row) => <TransferCell job={row.RowKey ? jobs[row.RowKey] : undefined}/>,
        },
        {
            title: "操作",
            width: 280,
            render: (_, row) => {
                const key = row.RowKey || ("id-" + row.Id);
                const job = jobs[key];
                if (row.Pending) {
                    return <>
                        {transferButtons(key, job)}
                        {(!job || job.phase === "error") &&
                            <Button type="link" danger onClick={() => removePending(key)}>删除</Button>}
                    </>;
                }
                const transferring = busy(key);
                return <>
                    {canPreview(row) &&
                        <Button type="link" onClick={() => setPreview(row)}>打开</Button>}
                    {row.FolderFlag != 1 && !transferring &&
                        <Button type="link" onClick={() => downloadFile(row)}>下载</Button>}
                    {transferButtons(key, job)}
                    <Button type="link" danger disabled={transferring} onClick={() => deleteIds([row.Id])}>删除</Button>
                </>;
            },
        },
    ];

    return (
        <div style={{maxWidth: 1200, padding: 24}}>
            <div style={{marginBottom: 16}}>
                <Link href="/users/profile">返回个人信息</Link>
            </div>
            <Breadcrumb style={{marginBottom: 16}}>
                <Breadcrumb.Item>
                    <a onClick={() => setParentId(0)}>根目录</a>
                </Breadcrumb.Item>
                {crumbs.map((c) => (
                    <Breadcrumb.Item key={c.Id}>
                        <a onClick={() => setParentId(c.Id)}>{c.Filename}</a>
                    </Breadcrumb.Item>
                ))}
            </Breadcrumb>
            <div style={{marginBottom: 16}}>
                <Radio.Group value={mode} onChange={(e) => setMode(e.target.value)} style={{marginRight: 16}}>
                    <Radio.Button value="basic">普通上传</Radio.Button>
                    <Radio.Button value="chunk">分片上传</Radio.Button>
                </Radio.Group>
                <Upload
                    showUploadList={false}
                    customRequest={(opt) => {
                        const file = opt.file as File;
                        Promise.resolve(startUpload(file))
                            .then(() => opt.onSuccess?.(null as any))
                            .catch((e) => opt.onError?.(e));
                    }}
                >
                    <Button type="primary">上传</Button>
                </Upload>
                &nbsp;&nbsp;
                <Button onClick={() => folderInputRef.current?.click()}>上传文件夹</Button>
                <input
                    ref={folderInputRef}
                    type="file"
                    multiple
                    style={{display: "none"}}
                    onChange={(e) => {
                        const list = e.target.files ? Array.from(e.target.files) : [];
                        e.target.value = "";
                        if (list.length > 0) {
                            uploadFolder(list);
                        }
                    }}
                />
                &nbsp;&nbsp;
                <Button onClick={() => setFolderOpen(true)}>新建文件夹</Button>
                {selected.length > 0 &&
                    <>
                        &nbsp;&nbsp;
                        <Button danger onClick={() => deleteIds(selected)}>批量删除</Button>
                    </>
                }
            </div>
            <Table
                rowKey={(row) => row.RowKey || String(row.Id)}
                loading={loading}
                columns={columns}
                dataSource={rows}
                rowSelection={{
                    selectedRowKeys: selected.map((id) => "id-" + id),
                    getCheckboxProps: (row) => ({disabled: !!row.Pending}),
                    onChange: (keys) => {
                        const ids = keys
                            .map((k) => Number(String(k).replace(/^id-/, "")))
                            .filter((id) => id > 0);
                        setSelected(ids);
                    },
                }}
                pagination={false}
            />
            <Modal
                title="新建文件夹"
                open={folderOpen}
                onOk={createFolder}
                onCancel={() => setFolderOpen(false)}
            >
                <Input
                    placeholder="文件夹名"
                    value={folderName}
                    onChange={(e) => setFolderName(e.target.value)}
                />
            </Modal>
            <FilePreview row={preview} onClose={() => setPreview(null)}/>
        </div>
    );
}

export default Page;
