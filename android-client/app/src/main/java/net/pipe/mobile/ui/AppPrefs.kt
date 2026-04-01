package net.pipe.mobile.ui

import android.content.Context

class AppPrefs(context: Context) {
    private val prefs = context.getSharedPreferences("pipe_mobile", Context.MODE_PRIVATE)

    fun endpoint(): String = prefs.getString(KEY_ENDPOINT, "http://10.0.2.2:8080").orEmpty()

    fun username(): String = prefs.getString(KEY_USERNAME, "").orEmpty()

    fun saveEndpoint(value: String) {
        prefs.edit().putString(KEY_ENDPOINT, value.trim()).apply()
    }

    fun saveUsername(value: String) {
        prefs.edit().putString(KEY_USERNAME, value.trim()).apply()
    }

    private companion object {
        const val KEY_ENDPOINT = "endpoint"
        const val KEY_USERNAME = "username"
    }
}
