package net.pipe.mobile.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.lifecycle.viewmodel.compose.viewModel
import net.pipe.mobile.data.MobileApiRepository
import net.pipe.mobile.ui.screens.ChatShell

@Composable
fun PipeApp() {
    val context = LocalContext.current
    val repository = remember { MobileApiRepository() }
    val prefs = remember { AppPrefs(context) }
    val viewModel: PipeViewModel = viewModel(factory = PipeViewModel.factory(repository, prefs))
    val colors = if (isSystemInDarkTheme()) {
        darkColorScheme(
            primary = Color(0xFF23CBFF),
            secondary = Color(0xFF8844FF),
            tertiary = Color(0xFF6E8BFF),
            background = Color(0xFF070B1C),
            surface = Color(0xFF101734),
            surfaceVariant = Color(0xFF162046),
            onBackground = Color(0xFFF6F7FF),
            onSurface = Color(0xFFF6F7FF),
            onSurfaceVariant = Color(0xFFB3BDE3),
            outline = Color(0xFF27335F),
            error = Color(0xFFFF6B88),
        )
    } else {
        lightColorScheme(
            primary = Color(0xFF127FE7),
            secondary = Color(0xFF6C37F4),
            tertiary = Color(0xFF18B7D9),
            background = Color(0xFFF1F5FF),
            surface = Color(0xFFFFFFFF),
            surfaceVariant = Color(0xFFE6EDFF),
            onBackground = Color(0xFF0B1020),
            onSurface = Color(0xFF0B1020),
            onSurfaceVariant = Color(0xFF4E5C82),
            outline = Color(0xFFD2DBF4),
            error = Color(0xFFD92D57),
        )
    }

    MaterialTheme(colorScheme = colors) {
        ChatShell(viewModel)
    }
}
