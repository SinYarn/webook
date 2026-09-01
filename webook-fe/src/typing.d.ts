declare const BACKEND_BASE_URL: 'http://localhost:8080';

type Profile = {
    Email: string
    Phone: string
    Nickname: string
    Birthday: string
    AboutMe: string
}

type UserFile = {
    Id: number
    ParentId: number
    Filename: string
    FolderFlag: number
    FileSizeDesc: string
    Utime: number
    FileSize?: number
    Pending?: boolean
    RowKey?: string
}

type FileBreadcrumb = {
    Id: number
    Filename: string
}

declare module 'spark-md5' {
    class SparkMD5 {
        append(str: string): void
        end(): string
    }
    namespace SparkMD5 {
        class ArrayBuffer {
            append(arr: globalThis.ArrayBuffer): void
            end(): string
        }
    }
    export default SparkMD5
}
