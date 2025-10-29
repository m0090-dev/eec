
using Avalonia;
using Avalonia.Controls;
using Avalonia.Interactivity;
using Avalonia.Media;
using Avalonia.Layout;
using Avalonia.Platform.Storage;
using Avalonia.Threading;
using System;
using System.Runtime.InteropServices;
using System.Threading.Tasks;
using System.Linq;
using System.IO;
using System.Collections.Generic;
using System.Diagnostics;
using System.Text.Json;
using MsBox.Avalonia;
using MsBox.Avalonia.Dto;
using MsBox.Avalonia.Enums;

namespace GEEC
{

    public class RunConfigSaveModel
    {
        public string Header { get; set; } = "";
        public string Program { get; set; } = "";
        public string ConfigFile { get; set; } = "";
        public string Tag { get; set; } = "";
        public int WaitTimeout { get; set; } = 60;
        public List<string> ProgramArgs { get; set; } = new();
        public List<string> Imports { get; set; } = new();
        public bool HideWindow { get; set; } = false;
    }



    public class RunConfigData
    {
        public Expander Expander { get; set; } = null!;
        public StackPanel Panel { get; set; } = null!;
        public TextBlock HeaderText { get; set; } = null!;
        public TextBox ProgramBox { get; set; } = null!;
        public TextBox ConfigFileBox { get; set; } = null!;
        public TextBox TagBox { get; set; } = null!;
        public TextBox WaitTimeoutBox { get; set; } = null!;
        public TextBox ArgInputBox { get; set; } = null!;
        public StackPanel ArgListPanel { get; set; } = null!;
        public List<string> ProgramArgs { get; set; } = new List<string>();
        public TextBox ImportInputBox { get; set; } = null!;
        public StackPanel ImportListPanel { get; set; } = null!;
        public List<string> Imports { get; set; } = new List<string>();
        public Button RunButton { get; set; } = null!;
        public Button DeleteButton { get; set; } = null!;
        public TextBox StdoutBox { get; set; } = null!;
        public TextBox StderrBox { get; set; } = null!;
        public CheckBox hideWindow { get; set; } = new CheckBox { Content = "HideWindow", IsChecked = false };
    }

    public partial class MainWindow : Window
    {
        private int configCounter = 0;
        private const string SaveFile = "runconfigs.json";
        private List<RunConfigData> configs = new List<RunConfigData>();
        public MainWindow()
        {
            InitializeComponent();
            this.Opened += async (_, __) => await RestoreConfigsIfAny();
        }

        private async Task RestoreConfigsIfAny() { if (File.Exists(SaveFile)) { var box = MessageBoxManager.GetMessageBoxStandard(new MessageBoxStandardParams { ContentTitle = "復元確認", ContentMessage = "前回の構成を復元しますか？", ButtonDefinitions = ButtonEnum.YesNo, Icon = MsBox.Avalonia.Enums.Icon.Question }); var result = await box.ShowWindowDialogAsync(this); if (result == ButtonResult.Yes) { try { var json = File.ReadAllText(SaveFile); var configs = JsonSerializer.Deserialize<List<RunConfigSaveModel>>(json); if (configs != null) { foreach (var cfg in configs) { CreateRunConfigFromSave(cfg); } } } catch (Exception ex) { Console.WriteLine($"復元エラー: {ex.Message}"); } } } }


        private void SaveConfigs()
        {
            var list = new List<RunConfigSaveModel>();
            foreach (var child in RunConfigList.Children)
            {
                if (child is Expander exp && exp.Content is StackPanel panel && exp.Tag is RunConfigData data)
                {
                    list.Add(new RunConfigSaveModel
                    {
                        Header = data.HeaderText.Text,
                        Program = data.ProgramBox.Text,
                        ConfigFile = data.ConfigFileBox.Text,
                        Tag = data.TagBox.Text,
                        WaitTimeout = int.TryParse(data.WaitTimeoutBox.Text, out var wt) ? wt : 60,
                        ProgramArgs = new List<string>(data.ProgramArgs),
                        Imports = new List<string>(data.Imports),
                        HideWindow = data.hideWindow.IsChecked ?? false
                    });
                }
            }

            File.WriteAllText(SaveFile,
                JsonSerializer.Serialize(list, new JsonSerializerOptions { WriteIndented = true }));
        }

        private void CreateRunConfigFromSave(RunConfigSaveModel cfg)
        {
            RunConfig_Create_Click(null, null);

            if (RunConfigList.Children.LastOrDefault() is Expander exp && exp.Tag is RunConfigData data)
            {
                data.HeaderText.Text = cfg.Header;
                data.ProgramBox.Text = cfg.Program;
                data.ConfigFileBox.Text = cfg.ConfigFile;
                data.TagBox.Text = cfg.Tag;
                data.WaitTimeoutBox.Text = cfg.WaitTimeout.ToString();
                data.hideWindow.IsChecked = cfg.HideWindow;

                foreach (var arg in cfg.ProgramArgs)
                {
                    data.ProgramArgs.Add(arg);
                    data.ArgListPanel.Children.Add(new TextBlock { Text = arg });
                }
                foreach (var imp in cfg.Imports)
                {
                    data.Imports.Add(imp);
                    data.ImportListPanel.Children.Add(new TextBlock { Text = imp });
                }
            }
        }




        private void RunConfig_Create_Click(object? sender, RoutedEventArgs e)
        {
            configCounter++;
            var data = new RunConfigData();

            // Expander & Header
            var expander = new Expander { IsExpanded = true };
            data.Expander = expander;

            var headerPanel = new StackPanel { Orientation = Orientation.Horizontal, VerticalAlignment = Avalonia.Layout.VerticalAlignment.Center };
            var headerText = new TextBlock { Text = $"構成名 {configCounter}" };
            data.HeaderText = headerText;

            var headerRunBtn = new Button { Content = "実行", Tag = data, Margin = new Thickness(5, 0, 0, 0) };
            var headerDelBtn = new Button { Content = "削除", Tag = data, Margin = new Thickness(5, 0, 0, 0) };
            headerRunBtn.Click += RunConfig_Run_Click;
            headerDelBtn.Click += RunConfig_Delete_Click;

            headerPanel.Children.Add(headerText);
            headerPanel.Children.Add(headerRunBtn);
            headerPanel.Children.Add(headerDelBtn);
            expander.Header = headerPanel;

            // HeaderTextクリックで編集
            headerText.PointerPressed += (s, _) =>
            {
                var tb = new TextBox { Text = headerText.Text, Width = 200 };
                tb.LostFocus += (s2, _) =>
                {
                    headerText.Text = tb.Text;
                    int index = headerPanel.Children.IndexOf(tb);
                    if (index >= 0) headerPanel.Children[index] = headerText;
                };
                tb.KeyDown += (s2, ke) =>
                {
                    if (ke.Key == Avalonia.Input.Key.Enter)
                    {
                        headerText.Text = tb.Text;
                        int index = headerPanel.Children.IndexOf(tb);
                        if (index >= 0) headerPanel.Children[index] = headerText;
                    }
                };
                int headerIndex = headerPanel.Children.IndexOf(headerText);
                if (headerIndex >= 0) headerPanel.Children[headerIndex] = tb;
                tb.Focus();
            };

            // Main Panel
            var panel = new StackPanel { Spacing = 5, Margin = new Thickness(0, 5, 0, 5) };
            data.Panel = panel;

            // Program
            panel.Children.Add(new TextBlock { Text = "プログラム:" });
            data.ProgramBox = new TextBox { Watermark = "実行するプログラム名", Width = 400 };
            panel.Children.Add(data.ProgramBox);

            // ConfigFile
            panel.Children.Add(new TextBlock { Text = "設定ファイル:" });
            var configPanel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
            data.ConfigFileBox = new TextBox { Watermark = "設定ファイルパス", Width = 400 };
            var configButton = new Button { Content = "参照", Tag = data };
            configButton.Click += SelectConfigFile_Click;
            configPanel.Children.Add(data.ConfigFileBox);
            configPanel.Children.Add(configButton);
            panel.Children.Add(configPanel);

            // tag
            panel.Children.Add(new TextBlock { Text = "tag:" });
            data.TagBox = new TextBox { Width = 200 };
            panel.Children.Add(data.TagBox);

            // waitTimeout
            panel.Children.Add(new TextBlock { Text = "waitTimeout (ms):" });
            data.WaitTimeoutBox = new TextBox { Text = "60000", Width = 100 };
            panel.Children.Add(data.WaitTimeoutBox);

            // ProgramArgs
            panel.Children.Add(new TextBlock { Text = "引数 (個別追加):" });
            var argsInputPanel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
            data.ArgInputBox = new TextBox { Width = 200 };
            var addArgBtn = new Button { Content = "Add", Tag = data };
            addArgBtn.Click += AddArg_Click;
            argsInputPanel.Children.Add(data.ArgInputBox);
            argsInputPanel.Children.Add(addArgBtn);
            data.ArgListPanel = new StackPanel { Spacing = 2 };
            panel.Children.Add(argsInputPanel);
            panel.Children.Add(data.ArgListPanel);

            // Imports
            panel.Children.Add(new TextBlock { Text = "Imports (個別追加):" });
            var importInputPanel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
            data.ImportInputBox = new TextBox { Width = 200 };
            var addImpBtn = new Button { Content = "Add", Tag = data };
            addImpBtn.Click += AddImport_Click;
            importInputPanel.Children.Add(data.ImportInputBox);
            importInputPanel.Children.Add(addImpBtn);
            data.ImportListPanel = new StackPanel { Spacing = 2 };
            panel.Children.Add(importInputPanel);
            panel.Children.Add(data.ImportListPanel);

            // Hide window
            var hideWindowPanel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
            hideWindowPanel.Children.Add(data.hideWindow);
            panel.Children.Add(hideWindowPanel);

            // stdout/stderr
            var outputPanel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
            data.StdoutBox = new TextBox { Width = 300, Height = 100, IsReadOnly = true, Background = Brushes.Black, Foreground = Brushes.Lime };
            data.StderrBox = new TextBox { Width = 300, Height = 100, IsReadOnly = true, Background = Brushes.Black, Foreground = Brushes.Red };
            outputPanel.Children.Add(data.StdoutBox);
            outputPanel.Children.Add(data.StderrBox);
            panel.Children.Add(outputPanel);

            // 下部 Run/Delete
            var buttonPanel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
            data.RunButton = new Button { Content = "実行", Tag = data };
            data.DeleteButton = new Button { Content = "削除", Tag = data };
            data.RunButton.Click += RunConfig_Run_Click;
            data.DeleteButton.Click += RunConfig_Delete_Click;
            buttonPanel.Children.Add(data.RunButton);
            buttonPanel.Children.Add(data.DeleteButton);
            panel.Children.Add(buttonPanel);

            expander.Content = panel;
            expander.Tag = data;
            RunConfigList?.Children.Add(expander);
            configs.Add(data);
        }


        private void AddArg_Click(object? sender, RoutedEventArgs e)
        {
            if (sender is Button btn && btn.Tag is RunConfigData data)
            {
                string arg = data.ArgInputBox.Text.Trim();
                if (!string.IsNullOrEmpty(arg))
                {
                    // 追加UI
                    var panel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
                    var tb = new TextBlock { Text = arg };
                    var delBtn = new Button { Content = "削除", Width = 50, Tag = (arg, data, panel) };

                    delBtn.Click += (s, _) =>
                    {
                        var (val, d, p) = ((string, RunConfigData, StackPanel))((Button)s!).Tag!;
                        d.ProgramArgs.Remove(val);
                        d.ArgListPanel.Children.Remove(p);
                    };

                    panel.Children.Add(tb);
                    panel.Children.Add(delBtn);

                    data.ProgramArgs.Add(arg);
                    data.ArgListPanel.Children.Add(panel);
                }
                data.ArgInputBox.Text = "";
            }
        }

        private void AddImport_Click(object? sender, RoutedEventArgs e)
        {
            if (sender is Button btn && btn.Tag is RunConfigData data)
            {
                string imp = data.ImportInputBox.Text.Trim();
                if (!string.IsNullOrEmpty(imp))
                {
                    var panel = new StackPanel { Orientation = Orientation.Horizontal, Spacing = 5 };
                    var tb = new TextBlock { Text = imp };
                    var delBtn = new Button { Content = "削除", Width = 50, Tag = (imp, data, panel) };

                    delBtn.Click += (s, _) =>
                    {
                        var (val, d, p) = ((string, RunConfigData, StackPanel))((Button)s!).Tag!;
                        d.Imports.Remove(val);
                        d.ImportListPanel.Children.Remove(p);
                    };

                    panel.Children.Add(tb);
                    panel.Children.Add(delBtn);

                    data.Imports.Add(imp);
                    data.ImportListPanel.Children.Add(panel);
                }
                data.ImportInputBox.Text = "";
            }
        }
        private async void RunConfig_Run_Click(object? sender, RoutedEventArgs e)
        {
            if (sender is Button btn && btn.Tag is RunConfigData data)
            {
                data.StdoutBox.Text = "";
                data.StderrBox.Text = "";

                string configFile = data.ConfigFileBox?.Text ?? "";
                string program = data.ProgramBox?.Text ?? "";
                string tag = data.TagBox?.Text ?? "";
                bool hideWindow = data.hideWindow.IsChecked ?? false;

                // wait-time-out を秒単位に
                int waitTimeoutSec = 60;
                if (!int.TryParse(data.WaitTimeoutBox?.Text, out waitTimeoutSec))
                    waitTimeoutSec = 60;

                string programArgs = string.Join(",", data.ProgramArgs.Select(arg => $"\"{arg}\""));
                string batPath = RuntimeInformation.IsOSPlatform(OSPlatform.Windows) ? @"eec.exe" : "eec";

                List<string> argsList = new List<string>();
                if (!string.IsNullOrWhiteSpace(configFile))
                    argsList.Add($"--config-file \"{configFile}\"");
                if (!string.IsNullOrWhiteSpace(program))
                    argsList.Add($"--program \"{program}\"");
                if (!string.IsNullOrWhiteSpace(programArgs))
                    argsList.Add($"--program-args={programArgs}");
                if (!string.IsNullOrWhiteSpace(tag))
                    argsList.Add($"--tag \"{tag}\"");
                if (hideWindow)
                    argsList.Add("--hide-window");
                argsList.Add($"--wait-time-out {waitTimeoutSec}");

                string arguments = "run " + string.Join(" ", argsList);

                await Task.Run(() =>
                {
                    try
                    {
                        if (hideWindow)
                        {
                            // GUI にリダイレクト
                            var psi = new ProcessStartInfo
                            {
                                FileName = batPath,
                                Arguments = arguments,
                                UseShellExecute = false,
                                RedirectStandardOutput = true,
                                RedirectStandardError = true,
                                RedirectStandardInput = true,
                                CreateNoWindow = true
                            };

                            Process proc = new Process { StartInfo = psi, EnableRaisingEvents = true };

                            proc.OutputDataReceived += (s, e2) =>
                            {
                                if (e2.Data != null)
                                    Dispatcher.UIThread.Post(() =>
                                        data.StdoutBox.Text += e2.Data + "\n"
                                    );
                            };

                            proc.ErrorDataReceived += (s, e2) =>
                            {
                                if (e2.Data != null)
                                    Dispatcher.UIThread.Post(() =>
                                        data.StderrBox.Text += e2.Data + "\n"
                                    );
                            };

                            proc.Start();
                            proc.BeginOutputReadLine();
                            proc.BeginErrorReadLine();
                            proc.WaitForExit();

                            Dispatcher.UIThread.Post(() =>
                                data.StdoutBox.Text += $"終了コード: {proc.ExitCode}\n"
                            );
                        }
                        else
                        {
                            // 通常コマンドプロンプトで実行
                            Process.Start(new ProcessStartInfo
                            {
                                FileName = "cmd.exe",
                                Arguments = $"/C \"{batPath} {arguments}\"",
                                UseShellExecute = true
                            });
                        }
                    }
                    catch (Exception ex)
                    {
                        Dispatcher.UIThread.Post(() =>
                        {
                            data.StderrBox.Text += $"Exception: {ex.Message}\n";
                        });
                    }
                });
            }
        }





        private void RunConfig_Delete_Click(object? sender, RoutedEventArgs e)
        {
            if (sender is Button btn && btn.Tag is RunConfigData data)
                RunConfigList?.Children.Remove(data.Expander);
            if (configCounter > 0)
                --configCounter;
        }

        private async void SelectConfigFile_Click(object? sender, RoutedEventArgs e)
        {
            if (sender is Button btn && btn.Tag is RunConfigData data && this.StorageProvider != null)
            {
                var files = await this.StorageProvider.OpenFilePickerAsync(new FilePickerOpenOptions
                {
                    Title = "設定ファイルを選択",
                    AllowMultiple = false,

                    FileTypeFilter = new List<FilePickerFileType>
{
    new FilePickerFileType("設定ファイル")
    {
        Patterns = new[] { "*.json", "*.yaml", "*.yml","*.toml"}
    },
    new FilePickerFileType("すべてのファイル")
    {
        Patterns = new[] { "*.*" }
    }
}
                });
                if (files != null && files.Any())
                    data.ConfigFileBox.Text = files[0].Path?.ToString() ?? "";
            }
        }

        private async void Quit_Click(object? sender, RoutedEventArgs e)
        {
            this.Close();
        }
        protected override void OnClosing(WindowClosingEventArgs e)
        {
            base.OnClosing(e);
            SaveConfigs();
        }

        // 空ハンドラ
        private void Tag_Add_Click(object? sender, RoutedEventArgs e) { }
        private void Tag_Remove_Click(object? sender, RoutedEventArgs e) { }
        private void Tag_List_Click(object? sender, RoutedEventArgs e) { }
        private void GenScript_Click(object? sender, RoutedEventArgs e) { }
    }
}
