import React, {useEffect, useState} from "react";
import axios from "@/axios/axios";

const imageExtensions = ["jpg", "jpeg", "png", "gif", "webp", "bmp", "svg"];

const labels: Record<string, {text: string; color: string}> = {
    folder: {text: "DIR", color: "#d89614"},
    video: {text: "VIDEO", color: "#722ed1"},
    audio: {text: "AUDIO", color: "#eb2f96"},
    pdf: {text: "PDF", color: "#cf1322"},
    word: {text: "WORD", color: "#0958d9"},
    excel: {text: "XLS", color: "#237804"},
    ppt: {text: "PPT", color: "#d4380d"},
    archive: {text: "ZIP", color: "#ad6800"},
    code: {text: "CODE", color: "#08979c"},
    text: {text: "TXT", color: "#595959"},
    file: {text: "FILE", color: "#434343"},
};

function extension(filename: string): string {
    const index = filename.lastIndexOf(".");
    return index < 0 ? "" : filename.slice(index + 1).toLowerCase();
}

function fileKind(file: UserFile): string {
    if (file.FolderFlag == 1) {
        return "folder";
    }
    const ext = extension(file.Filename);
    if (imageExtensions.includes(ext)) {
        return "image";
    }
    if (["mp4", "webm", "mov", "mkv", "avi"].includes(ext)) {
        return "video";
    }
    if (["mp3", "wav", "flac", "aac", "m4a"].includes(ext)) {
        return "audio";
    }
    if (ext == "pdf") {
        return "pdf";
    }
    if (["doc", "docx"].includes(ext)) {
        return "word";
    }
    if (["xls", "xlsx", "csv"].includes(ext)) {
        return "excel";
    }
    if (["ppt", "pptx"].includes(ext)) {
        return "ppt";
    }
    if (["zip", "rar", "7z", "tar", "gz"].includes(ext)) {
        return "archive";
    }
    if (["js", "ts", "tsx", "jsx", "go", "py", "java", "c", "cpp", "html", "css", "json", "sql"].includes(ext)) {
        return "code";
    }
    if (["txt", "md", "log"].includes(ext)) {
        return "text";
    }
    return "file";
}

function imageMime(filename: string): string {
    const ext = extension(filename);
    if (ext == "jpg" || ext == "jpeg") {
        return "image/jpeg";
    }
    if (ext == "svg") {
        return "image/svg+xml";
    }
    return `image/${ext || "png"}`;
}

const thumbStyle: React.CSSProperties = {
    width: 44,
    height: 44,
    borderRadius: 6,
    objectFit: "cover",
    flexShrink: 0,
};

function FileThumb({row = {
    Id: 0,
    ParentId: 0,
    Filename: "",
    FolderFlag: 0,
    FileSizeDesc: "",
    Utime: 0,
}}: {row?: UserFile}) {
    const kind = fileKind(row);
    const [preview, setPreview] = useState("");

    useEffect(() => {
        setPreview("");
        if (kind != "image") {
            return;
        }
        let objectUrl = "";
        let cancelled = false;
        axios.get("/files/download", {
            params: {id: row.Id},
            responseType: "blob",
        }).then((res) => {
            if (cancelled || res.status != 200) {
                return;
            }
            const blob = new Blob([res.data], {type: imageMime(row.Filename)});
            objectUrl = URL.createObjectURL(blob);
            setPreview(objectUrl);
        }).catch(() => {
        });
        return () => {
            cancelled = true;
            if (objectUrl) {
                URL.revokeObjectURL(objectUrl);
            }
        };
    }, [kind, row.Filename, row.Id]);

    if (preview) {
        return <img alt={row.Filename} src={preview} style={thumbStyle}/>;
    }

    const label = labels[kind] || labels.file;
    return <div style={{
        ...thumbStyle,
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        background: label.color,
        color: "#fff",
        fontSize: 10,
        fontWeight: 700,
    }}>
        {label.text}
    </div>;
}

export default FileThumb;
