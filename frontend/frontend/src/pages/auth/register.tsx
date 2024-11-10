
import { BetterForm } from "~/components/customComponentsNotFromShadcn/betterForm"
import { type pageProps } from "../_app"
import { PopUpFormWrapper } from "~/components/customComponentsNotFromShadcn/hideableCompoent"
import { useState } from "react"
import { cx } from "class-variance-authority"
import { redirectTo } from "~/utils/redirector"

const Signup: React.FC<pageProps> = ({ ctx }) => {
    const [form, setForm] = useState({
        username: "",
        password: "",
        confirmPassword: "",
    })
    const [error, setError] = useState<string | null>(null)

    const handleSignup = async () => {
        // Check if password and confirmPassword match
        if (form.password !== form.confirmPassword) {
            setError("Passwords do not match")
            return
        }

        try {
            const res = await ctx.api.signup(form.username, form.password)
            if (res.status < 300) {
                // Optionally set token if backend provides it, then redirect
                redirectTo("/auth/login")
            }
        } catch (err) {
            setError("Error signing up. Please try again.")
            console.log("err",err)
        }
    }

    return (
        <div className="flex h-screen items-center justify-center bg-gray-100">
            <div className="w-full max-w-md p-6 bg-white rounded-lg shadow-md">
                <h1 className="text-2xl font-semibold text-center text-gray-800 mb-6">Signup</h1>
                <PopUpFormWrapper isHidden={false} onSubmit={() => {}}>
                    <div className="space-y-4">
                        {error && <p className="text-red-500 text-center">{error}</p>}
                        <input
                            type="text"
                            placeholder="Username"
                            value={form.username}
                            onChange={(e) => setForm({ ...form, username: e.target.value })}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        />
                        <input
                            type="password"
                            placeholder="Password"
                            value={form.password}
                            onChange={(e) => setForm({ ...form, password: e.target.value })}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        />
                        <input
                            type="password"
                            placeholder="Confirm Password"
                            value={form.confirmPassword}
                            onChange={(e) => setForm({ ...form, confirmPassword: e.target.value })}
                            className="w-full px-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-indigo-500"
                        />
                    </div>
                    <button
                        type="button"
                        onClick={handleSignup}
                        className="w-full mt-6 px-4 py-2 text-white bg-indigo-600 rounded-lg hover:bg-indigo-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:ring-offset-2"
                    >
                        Signup
                    </button>
                </PopUpFormWrapper>
            </div>
        </div>
    )
}

export default Signup
