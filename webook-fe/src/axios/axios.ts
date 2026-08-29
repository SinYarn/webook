import axios from "axios";
import router from "next/router";

const defaultBaseURL = typeof window === "undefined"
    ? "http://localhost:8080"
    : `${window.location.protocol}//${window.location.hostname}:8080`

const instance = axios.create({
    baseURL: process.env.NEXT_PUBLIC_API_BASE_URL || defaultBaseURL,
    withCredentials: true
})


instance.interceptors.response.use(function (resp) {
    const newToken = resp.headers["x-jwt-token"]
    const newRefreshToken = resp.headers["x-refresh-token"]
    console.log("resp headers", resp.headers)
    if (newToken) {
        localStorage.setItem("token", newToken)
    }
    if (newRefreshToken) {
        localStorage.setItem("refresh_token", newRefreshToken)
    }
    if (resp.status == 401) {
        window.location.href="/users/login"
    }
    return resp
}, (err) => {
    console.log(err)
    if (err.response?.status == 401) {
        window.location.href="/users/login"
    }
    return Promise.reject(err)
})

// 在这里让每一个请求都加上 authorization 的头部
instance.interceptors.request.use((req) => {
    const token = localStorage.getItem("token")
    req.headers.setAuthorization("Bearer " + token, true)
    return req
}, (err) => {
    console.log(err)
})

export default instance