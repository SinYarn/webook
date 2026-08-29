import React, {useEffect, useState} from 'react';
import {Breadcrumb, Button, Input, Modal, Radio, Table, Upload} from 'antd';
import type {ColumnsType} from 'antd/es/table';
import axios from "@/axios/axios";
import Link from "next/link";
import moment from "moment";
import SparkMD5 from "spark-md5";
import FileThumb from "./FileThumb";

const CHUNK_SIZE = 2 * 1024 * 1024;

function md5File(file: File): Promise<string> {
    return new Promise((resolve, reject) => {
        const chunks = Math.ceil(file.size / CHUNK_SIZE) || 1;
        const spark = new SparkMD5.ArrayBuffer();
        const reader = new FileReader();
        let i = 0;
        reader.onload = (e) => {
            spark.append(e.target!.result as ArrayBuffer);
            i++;
            if (i < chunks) {
                loadNext();
            } else {
                resolve(spark.end());
            }
        };
        reader.onerror = reject;
        const loadNext = () => {
            const start = i * CHUNK_SIZE;
            const end = Math.min(start + CHUNK_SIZE, file.size);
            reader.readAsArrayBuffer(file.slice(start, end));
        };
        loadNext();
    });
}

function Page() {
    const [parentId, setParentId] = useState(0);
    const [list, setList] = useState<UserFile[]>([]);
    const [crumbs, setCrumbs] = useState<FileBreadcrumb[]>([]);
    const [loading, setLoading] = useState(false);
    const [mode, setMode] = useState("basic");
    const [folderOpen, setFolderOpen] = useState(false);
    const [folderName, setFolderName] = useState("");
    const [selected, setSelected] = useState<number[]>([]);

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

    const basicUpload = (file: File) => {
        const fd = new FormData();
        fd.append("file", file);
        fd.append("parentId", String(parentId));
        return axios.post("/files/upload", fd).then((res) => {
            if (res.status != 200) {
                alert(res.statusText);
                return;
            }
            alert(res.data);
            load(parentId);
        }).catch((err) => {
            alert(err);
        });
    };

    const chunkUpload = async (file: File) => {
        try {
            const identifier = await md5File(file);
            const sec = await axios.post("/files/sec-upload", {
                parentId,
                filename: file.name,
                identifier,
            });
            if (sec.status != 200) {
                alert(sec.statusText);
                return;
            }
            if (sec.data === "秒传成功") {
                alert(sec.data);
                load(parentId);
                return;
            }
            if (sec.data !== "文件不存在") {
                alert(sec.data);
                return;
            }
            const totalChunks = Math.ceil(file.size / CHUNK_SIZE) || 1;
            const uploadedRes = await axios.get("/files/chunk-upload", {params: {identifier}});
            const uploaded: number[] = uploadedRes.data?.UploadedChunks || [];
            for (let i = 1; i <= totalChunks; i++) {
                if (uploaded.indexOf(i) >= 0) {
                    continue;
                }
                const start = (i - 1) * CHUNK_SIZE;
                const end = Math.min(start + CHUNK_SIZE, file.size);
                const blob = file.slice(start, end);
                const fd = new FormData();
                fd.append("file", blob, file.name);
                fd.append("filename", file.name);
                fd.append("identifier", identifier);
                fd.append("totalChunks", String(totalChunks));
                fd.append("chunkNumber", String(i));
                fd.append("totalSize", String(file.size));
                const r = await axios.post("/files/chunk-upload", fd);
                if (r.status != 200) {
                    alert(r.statusText);
                    return;
                }
                if (typeof r.data === "string") {
                    alert(r.data);
                    return;
                }
            }
            const merge = await axios.post("/files/merge", {
                parentId,
                filename: file.name,
                identifier,
                totalSize: file.size,
            });
            if (merge.status != 200) {
                alert(merge.statusText);
                return;
            }
            alert(merge.data);
            load(parentId);
        } catch (err) {
            alert(err);
        }
    };

    const downloadFile = (row: UserFile) => {
        axios.get("/files/download", {
            params: {id: row.Id},
            responseType: "blob",
        }).then((res) => {
            if (res.status != 200) {
                alert(res.statusText);
                return;
            }
            const blob = res.data as Blob;
            if (blob.type && blob.type.indexOf("text") >= 0) {
                blob.text().then((t) => alert(t));
                return;
            }
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement("a");
            a.href = url;
            a.download = row.Filename;
            document.body.appendChild(a);
            a.click();
            a.remove();
            window.URL.revokeObjectURL(url);
        }).catch((err) => {
            alert(err);
        });
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
            alert(res.data);
            setFolderOpen(false);
            setFolderName("");
            load(parentId);
        }).catch((err) => {
            alert(err);
        });
    };

    const columns: ColumnsType<UserFile> = [
        {
            title: "文件名",
            dataIndex: "Filename",
            render: (_, row) => {
                const name = row.FolderFlag == 1
                    ? <a onClick={() => setParentId(row.Id)}>{row.Filename}</a>
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
            title: "操作",
            width: 160,
            render: (_, row) => {
                return <>
                    {row.FolderFlag != 1 &&
                        <Button type="link" onClick={() => downloadFile(row)}>下载</Button>}
                    <Button type="link" danger onClick={() => deleteIds([row.Id])}>删除</Button>
                </>;
            },
        },
    ];

    return (
        <div style={{maxWidth: 960, padding: 24}}>
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
                        const p = mode === "chunk" ? chunkUpload(file) : basicUpload(file);
                        Promise.resolve(p).then(() => opt.onSuccess?.(null as any)).catch((e) => opt.onError?.(e));
                    }}
                >
                    <Button type="primary">上传</Button>
                </Upload>
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
                rowKey="Id"
                loading={loading}
                columns={columns}
                dataSource={list}
                rowSelection={{
                    selectedRowKeys: selected,
                    onChange: (keys) => setSelected(keys as number[]),
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
        </div>
    );
}

export default Page;
