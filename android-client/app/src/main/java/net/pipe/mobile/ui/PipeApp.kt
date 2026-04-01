package net.pipe.mobile.ui

import androidx.compose.foundation.isSystemInDarkTheme
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.darkColorScheme
import androidx.compose.material3.lightColorScheme
import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
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
            primary = androidx.compose.ui.graphics.Color(0xFFE7A977),
            secondary = androidx.compose.ui.graphics.Color(0xFFA9C8A2),
            background = androidx.compose.ui.graphics.Color(0xFF101A26),
            surface = androidx.compose.ui.graphics.Color(0xFF152232),
        )
    } else {
        lightColorScheme(
            primary = androidx.compose.ui.graphics.Color(0xFF9A4F24),
            secondary = androidx.compose.ui.graphics.Color(0xFF45674E),
            background = androidx.compose.ui.graphics.Color(0xFFF3EEE6),
            surface = androidx.compose.ui.graphics.Color(0xFFF8F4EE),
        )
    }

    MaterialTheme(colorScheme = colors) {
        ChatShell(viewModel)
    }
}
