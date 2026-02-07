// 平台特定 API

// 获取用户合集列表（视频号）
export async function getCollections(platform: string): Promise<{ label: string; value: string }[]> {
  try {
    // TODO: 实现获取合集列表功能
    console.log('获取合集列表:', platform)
    return []
  } catch (error) {
    console.error('获取合集列表失败:', error)
    return []
  }
}

// 自动选择推荐封面（抖音）
export async function autoSelectCover(videoId: number): Promise<{ thumbnailPath: string }> {
  try {
    // TODO: 实现自动选择推荐封面功能
    console.log('自动选择封面:', videoId)
    return { thumbnailPath: '' }
  } catch (error) {
    console.error('自动选择封面失败:', error)
    throw error
  }
}

// 验证商品链接（抖音）
export async function validateProductLink(link: string): Promise<{ valid: boolean; title?: string; error?: string }> {
  try {
    // TODO: 实现验证商品链接功能
    console.log('验证商品链接:', link)
    return { valid: false, error: '未实现' }
  } catch (error) {
    console.error('验证商品链接失败:', error)
    throw error
  }
}

// 选择图片文件
export async function selectImageFile(): Promise<string> {
  try {
    // TODO: 实现选择图片文件功能
    console.log('选择图片文件')
    return ''
  } catch (error) {
    console.error('选择图片失败:', error)
    throw error
  }
}

// 选择文件
export async function selectFile(accept?: string): Promise<string> {
  try {
    // TODO: 实现选择文件功能
    console.log('选择文件:', accept)
    return ''
  } catch (error) {
    console.error('选择文件失败:', error)
    throw error
  }
}
