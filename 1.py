import os
import pathlib
def get_language_by_extension(extension):
    """根据文件扩展名返回 Markdown 语言标识符"""
    mapping = {
        ".py": "python",
        ".js": "javascript",
        ".ts": "typescript",
        ".java": "java",
        ".c": "c",
        ".cpp": "cpp",
        ".cs": "csharp",
        ".go": "go",
        ".rb": "ruby",
        ".php": "php",
        ".html": "html",
        ".css": "css",
        ".json": "json",
        ".xml": "xml",
        ".md": "markdown",
        ".sh": "bash",
        ".yaml": "yaml",
        ".yml": "yaml",
        ".kt": "kotlin",
        ".swift": "swift",
        # 可以根据需要添加更多映射
    }
    return mapping.get(extension.lower(), "")
def directory_to_txt(directory_path, output_filename="output.txt"):
    """
    遍历指定目录下的所有文件，将文件名和内容写入一个 TXT 文件。
    文件内容用 Markdown 代码块包裹，语言根据文件后缀判断。
    :param directory_path: 要遍历的目录路径
    :param output_filename: 输出的 TXT 文件名
    """
    if not os.path.isdir(directory_path):
        print(f"错误：提供的路径 '{directory_path}' 不是一个有效的目录。")
        return
    with open(output_filename, "w", encoding="utf-8") as outfile:
        for root, _, files in os.walk(directory_path):
            for filename in files:
                file_path = pathlib.Path(root) / filename
                try:
                    with open(file_path, "r", encoding="utf-8") as infile:
                        content = infile.read()
                    relative_path = file_path.relative_to(directory_path)
                    outfile.write(f"## 文件名：{relative_path}\n\n")
                    extension = file_path.suffix
                    language = get_language_by_extension(extension)
                    outfile.write(f"```{language}\n")
                    outfile.write(content)
                    outfile.write("\n```\n\n")
                    outfile.write("---\n\n") # 添加分隔符
                except Exception as e:
                    outfile.write(f"## 文件名：{relative_path}\n\n")
                    outfile.write(f"无法读取文件内容或处理文件时出错：{e}\n\n")
                    outfile.write("---\n\n")
    print(f"处理完成！输出已保存到 '{output_filename}'")
if __name__ == "__main__":
    target_directory = input("请输入要处理的目录路径：")
    directory_to_txt(target_directory)